create table games
(
    id                   BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    current_player_order TINYINT UNSIGNED,
    board_base           TINYBLOB,
    board_positioning    TINYBLOB,
    state                TINYINT UNSIGNED
);
