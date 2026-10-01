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
