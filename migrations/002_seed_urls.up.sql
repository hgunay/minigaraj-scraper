-- ============================================================================
-- MiniGaraj Scraper - Seed URLs
-- ============================================================================
-- Dynamic seed URL management — replaces hardcoded URL lists in crawlers.
--
-- Author: Hakan Gunay
-- Date: 2026-04-04
-- ============================================================================

CREATE TABLE IF NOT EXISTS scraper.seed_urls (
    id              BIGSERIAL PRIMARY KEY,
    brand           VARCHAR(100)    NOT NULL,
    url             TEXT            NOT NULL,
    label           VARCHAR(255),                           -- "2024 mainline list", "Car Culture"
    category        VARCHAR(100),                           -- year_list, premium_line, category_page
    is_active       BOOLEAN         NOT NULL DEFAULT true,
    priority        INT             NOT NULL DEFAULT 0,     -- higher = crawl first
    last_crawled_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE(brand, url)
);

CREATE INDEX IF NOT EXISTS idx_seed_urls_brand_active ON scraper.seed_urls(brand, is_active);

-- ============================================================================
-- Insert default Hot Wheels seed URLs
-- ============================================================================
INSERT INTO scraper.seed_urls (brand, url, label, category, priority) VALUES
    -- Year lists
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2024_Hot_Wheels', '2024 Mainline', 'year_list', 10),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2023_Hot_Wheels', '2023 Mainline', 'year_list', 9),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2022_Hot_Wheels', '2022 Mainline', 'year_list', 8),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2021_Hot_Wheels', '2021 Mainline', 'year_list', 7),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2020_Hot_Wheels', '2020 Mainline', 'year_list', 6),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2019_Hot_Wheels', '2019 Mainline', 'year_list', 5),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2018_Hot_Wheels', '2018 Mainline', 'year_list', 4),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2017_Hot_Wheels', '2017 Mainline', 'year_list', 3),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2016_Hot_Wheels', '2016 Mainline', 'year_list', 2),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2015_Hot_Wheels', '2015 Mainline', 'year_list', 1),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2014_Hot_Wheels', '2014 Mainline', 'year_list', 0),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2013_Hot_Wheels', '2013 Mainline', 'year_list', 0),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2012_Hot_Wheels', '2012 Mainline', 'year_list', 0),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2011_Hot_Wheels', '2011 Mainline', 'year_list', 0),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/List_of_2010_Hot_Wheels', '2010 Mainline', 'year_list', 0),
    -- Premium lines
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/Car_Culture',           'Car Culture',   'premium_line', 8),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/Hot_Wheels_Premium',     'Premium',       'premium_line', 8),
    ('Hot Wheels', 'https://hotwheels.fandom.com/wiki/RLC_(Red_Line_Club)',    'Red Line Club', 'premium_line', 7)
ON CONFLICT (brand, url) DO NOTHING;
