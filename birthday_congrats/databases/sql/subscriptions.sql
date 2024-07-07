DROP TABLE IF EXISTS `subscriptions`;
CREATE TABLE `subscriptions` (
  `subscriber_id` int NOT NULL,
  `subscription_id` int NOT NULL,
  `days_alert` int NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;