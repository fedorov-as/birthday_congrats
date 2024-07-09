DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `username` text NOT NULL COLLATE utf8_bin,
  `password` text NOT NULL COLLATE utf8_bin,
  `email` text NOT NULL,
  `year` int NOT NULL,
  `month` int NOT NULL,
  `day` int NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO `users` (`id`, `username`, `password`, `email`, `year`, `month`, `day`) VALUES
(1,	'sasha',	'12345678', 'sashafe5555@gmail.com', 2000, 6, 30),
(2,	'admin',	'admin123', 'admin@123.ru', 1970, 1, 1);