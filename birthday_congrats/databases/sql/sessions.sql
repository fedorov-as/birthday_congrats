DROP TABLE IF EXISTS `sessions`;
CREATE TABLE `sessions` (
  `sess_id` text NOT NULL,
  `user_id` int NOT NULL,
  `expires` bigint
) ENGINE=InnoDB DEFAULT CHARSET=utf8;