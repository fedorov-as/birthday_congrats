DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` int(32) NOT NULL AUTO_INCREMENT,
  `username` text NOT NULL,
  `password` text NOT NULL,
  `email` text NOT NULL,
  `year` int(32) NOT NULL,
  `month` int(32) NOT NULL,
  `day` int(32) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO `users` (`id`, `username`, `password`, `email`, `year`, `month`, `day`) VALUES
(1,	'sasha',	'12345678', 'sashafe5555@gmail.com', 2000, 6, 30),
(2,	'admin',	'admin123', 'admin@123.ru', 1970, 1, 1);