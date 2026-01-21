-- hermet_0
-- 以 jrmarcco@gmail.com 为 sharder 插入数据到分表
INSERT INTO biz_user_2 (
    id,
    email,
    mobile,
    passwd,
    nickname,
    avatar,
    gender,
    tagline,
    region,
    birthday,
    info_ver,
    user_status,
    deleted_at,
    created_at,
    updated_at
) VALUES (
    134605228232032256,
    'jrmarcco@gmail.com',
    '13100131000',
    '$2a$10$besICPqbCRWOocqlsaKXV.rniGRyCNPLHeFT.osXbhgisW4XSW/um',
    'jrmarcco',
    'https://avatars.githubusercontent.com/u/jrmarcco',
    'male',
    'Stay hungry, stay foolish.',
    '',
    0,
    1,
    'active',
    0,
    EXTRACT(EPOCH FROM NOW()) * 1000,
    EXTRACT(EPOCH FROM NOW()) * 1000
);
