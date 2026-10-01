CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  employee_code bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
  name text NOT NULL,
  email text NOT NULL,
  phone text NOT NULL DEFAULT '',
  login text NOT NULL,
  password_hash text NOT NULL,
  role text NOT NULL DEFAULT 'operator' CHECK (role IN ('admin','operator','finance','viewer')),
  permissions jsonb NOT NULL DEFAULT '[]'::jsonb,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
  last_seen_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_lower_uidx ON users(lower(email));
CREATE UNIQUE INDEX users_login_lower_uidx ON users(lower(login));
CREATE TRIGGER users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE refresh_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  user_agent text NOT NULL DEFAULT '',
  ip_address inet,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens(user_id);

CREATE TABLE customers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_code bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
  name text NOT NULL,
  phone text NOT NULL,
  email text NOT NULL DEFAULT '',
  country text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive','vip')),
  notes text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE UNIQUE INDEX customers_phone_uidx ON customers(phone) WHERE deleted_at IS NULL;
CREATE INDEX customers_search_idx ON customers USING gin(to_tsvector('simple', name || ' ' || phone || ' ' || email));
CREATE TRIGGER customers_updated_at BEFORE UPDATE ON customers FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE suppliers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  supplier_code bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
  name text NOT NULL UNIQUE,
  contact_name text NOT NULL DEFAULT '',
  contact_email text NOT NULL DEFAULT '',
  contact_phone text NOT NULL DEFAULT '',
  api_base_url text NOT NULL DEFAULT '',
  api_key_encrypted text NOT NULL DEFAULT '',
  api_status text NOT NULL DEFAULT 'offline' CHECK (api_status IN ('online','offline','warning')),
  currency char(3) NOT NULL DEFAULT 'KZT',
  deposit numeric(16,2) NOT NULL DEFAULT 0 CHECK (deposit >= 0),
  balance numeric(16,2) NOT NULL DEFAULT 0 CHECK (balance >= 0),
  terms text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled','low_balance')),
  low_balance_threshold numeric(16,2) NOT NULL DEFAULT 100000 CHECK (low_balance_threshold >= 0),
  last_synced_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE TRIGGER suppliers_updated_at BEFORE UPDATE ON suppliers FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE tariffs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tariff_code bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
  supplier_id uuid NOT NULL REFERENCES suppliers(id),
  external_id text NOT NULL DEFAULT '',
  name text NOT NULL,
  country text NOT NULL,
  region text NOT NULL DEFAULT '',
  data_mb bigint NOT NULL CHECK (data_mb > 0),
  validity_days integer NOT NULL CHECK (validity_days > 0),
  cost numeric(16,2) NOT NULL CHECK (cost >= 0),
  price numeric(16,2) NOT NULL CHECK (price >= 0),
  min_price numeric(16,2) NOT NULL DEFAULT 0 CHECK (min_price >= 0),
  recommended_price numeric(16,2) NOT NULL DEFAULT 0 CHECK (recommended_price >= 0),
  currency char(3) NOT NULL DEFAULT 'KZT',
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  UNIQUE(supplier_id, external_id)
);
CREATE INDEX tariffs_country_idx ON tariffs(country) WHERE deleted_at IS NULL;
CREATE TRIGGER tariffs_updated_at BEFORE UPDATE ON tariffs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE esims (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  esim_code bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
  supplier_id uuid NOT NULL REFERENCES suppliers(id),
  tariff_id uuid REFERENCES tariffs(id),
  iccid text NOT NULL UNIQUE,
  imsi text NOT NULL DEFAULT '',
  activation_code text NOT NULL DEFAULT '',
  smdp_address text NOT NULL DEFAULT '',
  qr_payload text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'available' CHECK (status IN ('available','reserved','sold','active','expired','disabled')),
  traffic_total_mb bigint NOT NULL DEFAULT 0 CHECK (traffic_total_mb >= 0),
  traffic_remaining_mb bigint NOT NULL DEFAULT 0 CHECK (traffic_remaining_mb >= 0),
  activated_at timestamptz,
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX esims_status_idx ON esims(status) WHERE deleted_at IS NULL;
CREATE INDEX esims_supplier_idx ON esims(supplier_id) WHERE deleted_at IS NULL;
CREATE TRIGGER esims_updated_at BEFORE UPDATE ON esims FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE orders (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  order_number bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
  customer_id uuid NOT NULL REFERENCES customers(id),
  tariff_id uuid NOT NULL REFERENCES tariffs(id),
  supplier_id uuid NOT NULL REFERENCES suppliers(id),
  esim_id uuid UNIQUE REFERENCES esims(id),
  operator_id uuid REFERENCES users(id) ON DELETE SET NULL,
  country text NOT NULL,
  plan_name text NOT NULL,
  cost numeric(16,2) NOT NULL CHECK (cost >= 0),
  price numeric(16,2) NOT NULL CHECK (price >= 0),
  fee numeric(16,2) NOT NULL DEFAULT 0 CHECK (fee >= 0),
  tax numeric(16,2) NOT NULL DEFAULT 0 CHECK (tax >= 0),
  profit numeric(16,2) GENERATED ALWAYS AS (price - cost - fee - tax) STORED,
  currency char(3) NOT NULL DEFAULT 'KZT',
  payment_method text NOT NULL DEFAULT 'cash' CHECK (payment_method IN ('cash','card','kaspi_qr','bank_transfer','other')),
  payment_status text NOT NULL DEFAULT 'pending' CHECK (payment_status IN ('pending','paid','refunded','failed')),
  status text NOT NULL DEFAULT 'created' CHECK (status IN ('created','pending_payment','paid','esim_assigned','activated','completed','cancelled','refunded')),
  activation_at timestamptz,
  expires_at timestamptz,
  cancelled_at timestamptz,
  cancellation_reason text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX orders_created_idx ON orders(created_at DESC);
CREATE INDEX orders_customer_idx ON orders(customer_id, created_at DESC);
CREATE INDEX orders_status_idx ON orders(status);
CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  type text NOT NULL,
  message text NOT NULL,
  level text NOT NULL DEFAULT 'info' CHECK (level IN ('info','warning','danger')),
  entity_type text NOT NULL DEFAULT '',
  entity_id uuid,
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_unread_idx ON notifications(created_at DESC) WHERE read_at IS NULL;

CREATE TABLE audit_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
  actor_name text NOT NULL,
  action text NOT NULL,
  entity_type text NOT NULL,
  entity_id uuid,
  old_value jsonb,
  new_value jsonb,
  ip_address inet,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_created_idx ON audit_logs(created_at DESC);
CREATE INDEX audit_logs_entity_idx ON audit_logs(entity_type, entity_id);

CREATE TABLE supplier_sync_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  supplier_id uuid NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
  status text NOT NULL CHECK (status IN ('running','success','failed')),
  tariffs_received integer NOT NULL DEFAULT 0,
  esims_received integer NOT NULL DEFAULT 0,
  error_message text NOT NULL DEFAULT '',
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz
);

