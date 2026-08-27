-- These codes have no supported city records and therefore cannot provide a
-- reliable longitude/timezone pair for birth-chart calculation. AN and CS are
-- also former country codes. Existing business references must be checked
-- before this migration is applied.
DELETE FROM geo_countries
WHERE code IN ('AN', 'AQ', 'BV', 'CS', 'HM', 'UM');
