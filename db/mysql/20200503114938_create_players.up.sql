create table players
(
    id                 BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    device_fingerprint VARCHAR(255) UNIQUE
);
