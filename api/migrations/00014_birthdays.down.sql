DROP TABLE settings;
DROP TABLE birthday_letters;
ALTER TABLE users DROP CONSTRAINT users_birthday_complete, DROP COLUMN birth_day, DROP COLUMN birth_month;
