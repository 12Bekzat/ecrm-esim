INSERT INTO suppliers(name, contact_name, api_status, currency, deposit, balance, terms, status, low_balance_threshold)
VALUES
  ('BC Telecom', 'Bek Connect', 'online', 'KZT', 1800000, 748000, 'Предоплата, автообновление', 'active', 100000),
  ('Joytel', 'Global SIM Desk', 'online', 'USD', 2400000, 325000, 'Постоплата 7 дней', 'active', 100000),
  ('Airalo Pro', 'Partner API', 'warning', 'USD', 1200000, 96000, 'API only', 'low_balance', 100000)
ON CONFLICT(name) DO NOTHING;

INSERT INTO tariffs(supplier_id, external_id, name, country, region, data_mb, validity_days, cost, price, min_price, recommended_price)
SELECT s.id, v.external_id, v.name, v.country, v.region, v.data_mb, v.validity_days, v.cost, v.price, v.min_price, v.recommended_price
FROM (VALUES
  ('BC Telecom','TR-LITE-3','Turkey Lite','Турция','TR',3072::bigint,7,1200::numeric,3500::numeric,3200::numeric,3900::numeric),
  ('Joytel','CN-TIKTOK-20','China TikTok','Китай','CN',20480::bigint,30,5600::numeric,8900::numeric,8200::numeric,9500::numeric),
  ('Airalo Pro','EU-33-10','Europe 33','Европа','EU',10240::bigint,15,4400::numeric,6900::numeric,6500::numeric,7400::numeric),
  ('BC Telecom','UAE-BIZ-10','UAE Business','ОАЭ','GCC',10240::bigint,15,6100::numeric,11500::numeric,10800::numeric,12400::numeric)
) AS v(supplier_name,external_id,name,country,region,data_mb,validity_days,cost,price,min_price,recommended_price)
JOIN suppliers s ON s.name=v.supplier_name
ON CONFLICT(supplier_id, external_id) DO NOTHING;

INSERT INTO esims(supplier_id, tariff_id, iccid, imsi, activation_code, smdp_address, qr_payload, status, traffic_total_mb, traffic_remaining_mb)
SELECT t.supplier_id, t.id,
       '8981' || lpad(t.tariff_code::text, 6, '0') || lpad(gs::text, 10, '0'),
       '401' || lpad(t.tariff_code::text, 5, '0') || lpad(gs::text, 7, '0'),
       encode(gen_random_bytes(12), 'hex'), 'smdp.example.net',
       'LPA:1$smdp.example.net$' || encode(gen_random_bytes(12), 'hex'),
       'available', t.data_mb, t.data_mb
FROM tariffs t
CROSS JOIN generate_series(1, 8) gs
ON CONFLICT(iccid) DO NOTHING;

INSERT INTO notifications(type, message, level)
VALUES
  ('Баланс', 'Airalo Pro ниже лимита 100 000 KZT', 'danger'),
  ('Склад', 'UAE Business: проверьте остаток eSIM', 'warning'),
  ('API', 'Справочники поставщиков готовы к синхронизации', 'info');
