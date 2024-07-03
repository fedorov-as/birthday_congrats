DROP TABLE IF EXISTS `sessions`;
CREATE TABLE `sessions` (
  `sess_id` text NOT NULL,
  `user_id` int(32) NOT NULL,
  `expires` int(64)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;