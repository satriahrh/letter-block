create table played_words
(
    game_id   BIGINT UNSIGNED,
    player_id BIGINT UNSIGNED,
    word      VARCHAR(255),
    primary key (game_id, word)
);