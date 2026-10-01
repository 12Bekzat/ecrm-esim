ALTER TABLE orders
  ADD COLUMN provider_order_id text,
  ADD COLUMN provider_idempotency_key text,
  ADD COLUMN provisioning_status text NOT NULL DEFAULT 'not_required'
    CHECK (provisioning_status IN ('not_required','pending','provisioned','failed')),
  ADD COLUMN provisioning_error text NOT NULL DEFAULT '',
  ADD COLUMN provider_payload jsonb;

CREATE UNIQUE INDEX orders_provider_order_uidx
  ON orders(supplier_id, provider_order_id)
  WHERE provider_order_id IS NOT NULL;
CREATE UNIQUE INDEX orders_provider_idempotency_uidx
  ON orders(provider_idempotency_key)
  WHERE provider_idempotency_key IS NOT NULL;

ALTER TABLE esims
  ADD COLUMN external_id text,
  ADD COLUMN install_url text NOT NULL DEFAULT '',
  ADD COLUMN ios_tap_link text NOT NULL DEFAULT '',
  ADD COLUMN android_tap_link text NOT NULL DEFAULT '',
  ADD COLUMN provider_payload jsonb;

CREATE UNIQUE INDEX esims_supplier_external_uidx
  ON esims(supplier_id, external_id)
  WHERE external_id IS NOT NULL;

INSERT INTO suppliers(name, contact_name, api_base_url, api_status, currency, terms, status, low_balance_threshold)
VALUES('Zesimo', 'my_key', 'https://zesimo.com', 'offline', 'USD', 'Bearer API · on-demand provisioning', 'active', 25)
ON CONFLICT(name) DO UPDATE SET
  contact_name=EXCLUDED.contact_name,
  api_base_url=EXCLUDED.api_base_url,
  currency=EXCLUDED.currency,
  terms=EXCLUDED.terms;
