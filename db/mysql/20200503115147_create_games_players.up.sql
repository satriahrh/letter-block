create table games_players
(
    game_id   BIGINT UNSIGNED,
    player_id BIGINT UNSIGNED,
    foreign key (game_id) references games (id),
    foreign key (player_id) references players (id),
    primary key (game_id, player_id)
);