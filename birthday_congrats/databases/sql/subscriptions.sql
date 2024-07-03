DROP TABLE IF EXISTS `subscriptions`;
CREATE TABLE `subscriptions` (
  `subscriber_id` int(32) NOT NULL,
  `subscription_id` int(32) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;