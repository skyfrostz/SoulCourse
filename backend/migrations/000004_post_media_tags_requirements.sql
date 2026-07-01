ALTER TABLE posts ADD COLUMN IF NOT EXISTS image_urls TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE posts ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}';

UPDATE posts
SET image_urls = ARRAY[
      'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80',
      'https://images.unsplash.com/photo-1522202176988-66273c2fd55f?auto=format&fit=crop&w=1200&q=80'
    ],
    tags = ARRAY['物化生','学习方法','学长经验']
WHERE title = '从学渣到班级前十：我的逆袭经验';

UPDATE posts
SET image_urls = ARRAY[
      'https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?auto=format&fit=crop&w=1200&q=80'
    ],
    tags = ARRAY['真实数据','浙江选科','物理化学']
WHERE title = '浙江2024选考目录：物化双选和不限选考几乎持平';

UPDATE posts
SET tags = ARRAY['真实数据','上海高考','专业组']
WHERE title = '上海2024：为什么物化双选专业组明显变多？';

UPDATE posts
SET tags = ARRAY['真实数据','甘肃高考','专业覆盖']
WHERE title = '甘肃2024选科要求：含物理专业过半，物化双选约45.98%';
