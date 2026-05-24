-- ============================================================================
-- MiniGaraj Scraper - Matchbox & Mini GT seed URLs
-- ============================================================================
-- Author: Hakan Gunay
-- Date: 2026-04-04
-- ============================================================================

INSERT INTO scraper.seed_urls (brand, url, label, category, priority) VALUES
    -- Matchbox: year lists
    ('Matchbox', 'https://matchbox.fandom.com/wiki/List_of_2024_Matchbox', '2024 Mainline', 'year_list', 10),
    ('Matchbox', 'https://matchbox.fandom.com/wiki/List_of_2023_Matchbox', '2023 Mainline', 'year_list', 9),
    ('Matchbox', 'https://matchbox.fandom.com/wiki/List_of_2022_Matchbox', '2022 Mainline', 'year_list', 8),
    ('Matchbox', 'https://matchbox.fandom.com/wiki/List_of_2021_Matchbox', '2021 Mainline', 'year_list', 7),
    ('Matchbox', 'https://matchbox.fandom.com/wiki/List_of_2020_Matchbox', '2020 Mainline', 'year_list', 6),
    -- Matchbox: collector lines
    ('Matchbox', 'https://matchbox.fandom.com/wiki/Category:Matchbox_Collectors', 'Collectors Category', 'category_page', 5),
    ('Matchbox', 'https://matchbox.fandom.com/wiki/Matchbox_Collectors', 'Collectors', 'premium_line', 5),
    -- Mini GT: official site collections
    ('Mini GT', 'https://www.minigt.com/collections/all',          'All Products',  'collection', 10),
    ('Mini GT', 'https://www.minigt.com/collections/1-64',         '1:64 Scale',    'collection', 9),
    ('Mini GT', 'https://www.minigt.com/collections/new-arrivals', 'New Arrivals',  'collection', 8)
ON CONFLICT (brand, url) DO NOTHING;
