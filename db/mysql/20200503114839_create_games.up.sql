create table games
(
    id                   BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    current_player_order TINYINT UNSIGNED,
    number_of_player     TINYINT UNSIGNED,
    board_base           TINYBLOB,
    board_positioning    TINYBLOB,
    state                TINYINT UNSIGNED
);
