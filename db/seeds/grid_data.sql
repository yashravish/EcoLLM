-- Baseline grid carbon intensity data (gCO2e/kWh) by region.
-- Source: Electricity Maps / IEA 2023 averages.
-- Refreshed hourly in production by the carbon-service.

INSERT INTO grid_carbon_intensity (region, intensity_gco2_per_kwh, source, updated_at)
VALUES
  ('US-EAST',    386.0, 'IEA-2023', NOW()),
  ('US-WEST',    210.0, 'IEA-2023', NOW()),
  ('US-CENTRAL', 480.0, 'IEA-2023', NOW()),
  ('EU-WEST',    275.0, 'IEA-2023', NOW()),
  ('EU-NORTH',    98.0, 'IEA-2023', NOW()),
  ('EU-EAST',    612.0, 'IEA-2023', NOW()),
  ('AP-EAST',    581.0, 'IEA-2023', NOW()),
  ('AP-SOUTH',   708.0, 'IEA-2023', NOW()),
  ('CA-CENTRAL', 130.0, 'IEA-2023', NOW()),
  ('AU-EAST',    656.0, 'IEA-2023', NOW())
ON CONFLICT (region) DO UPDATE
  SET intensity_gco2_per_kwh = EXCLUDED.intensity_gco2_per_kwh,
      updated_at = NOW();
