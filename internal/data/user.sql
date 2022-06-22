-- drop database
DROP DATABASE IF EXISTS `goim`;

-- create database
create database if not exists goim;

-- define user table based on go structure User in current directory
DROP TABLE IF EXISTS goim.user;

create table if not exists goim.user (
	`id` bigint not null auto_increment,
	`uid` varchar(64) not null, -- 22 bytes of uuid
	`name` varchar(32) not null,
	`password` varchar(128) not null,
	`email` varchar(32),
	`phone` varchar(32),
	`avatar` varchar(128) not null,
	`status` tinyint not null DEFAULT 0,
	`created_at` int not null DEFAULT 0,
	`updated_at` int not null DEFAULT 0,
	primary key (`id`),
	unique key (`uid`),
    UNIQUE KEY (`email`),
    UNIQUE KEY (`phone`)
) auto_increment = 10000 engine = innodb charset = utf8mb4;

-- mock data
insert into goim.user (`id`, `uid`, `name`, `password`, `email`, `phone`, `avatar`, `status`, `created_at`, `updated_at`)
values
    (10000, '4F8DSQByUsEUMoETzTCabh', 'user1', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user1@example.com', NULL, ' ', 1, 1528894200, 1528894200),
    (10001, 'C6CtUjpC6h5e5SW9tBFNVX', 'user2', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user2@example.com', NULL, ' ', 0, 1528894200, 1528894200),
    (10002, '7mRZLYedtK1EwxzC5X1Lxf', 'user3', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user3@example.com', NULL, ' ', 0, 1528894200, 1528894200),
    (10003, 'WmbtshDDMUgb3KWFisWZ4E', 'user4', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user4@example.com', NULL, ' ', 0, 1528894200, 1528894200),
    (10004, 'Vf4gA6vQdeF81YHV7DU4pP', 'user5', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user5@example.com', NULL, ' ', 0, 1528894200, 1528894200),
    (10005, 'Pzu74cyA3BJhnj1fx2oSuz', 'user6', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user6@example.com', NULL, ' ', 0, 1528894200, 1528894200),
    (10006, 'KWZs8sLE1dNQRCscx4rs3q', 'user7', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user7@example.com', NULL, ' ', 0, 1528894200, 1528894200),
    (10007, 'KmFExCJdsVJ2ws8uZzg49d', 'user8', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user8@example.com', NULL, ' ', 0, 1528894200, 1528894200),
    (10008, 'URM38EZ2A1LA3qkyLuoS3D', 'user9', '8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92', 'user9@example.com', NULL, ' ', 0, 1528894200, 1528894200);

-- define friend table based on go structure Friend in current directory
DROP TABLE IF EXISTS goim.friend;

CREATE TABLE IF NOT EXISTS goim.friend (
    `id` bigint not null auto_increment,
    `uid` varchar(64) not null, -- 22 bytes of uuid
    `friend_uid` varchar(64) not null, -- 22 bytes of uuid
    `status` tinyint not null default 0 COMMENT '0: friend; 1: stranger; 2: blacked',
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`uid`, `friend_uid`) COMMENT 'uid and friend_uid are unique'
) auto_increment = 10000 engine = innodb charset = utf8mb4;

-- define friend_request table based on go structure FriendRequest in current directory
DROP TABLE IF EXISTS goim.friend_request;

CREATE TABLE IF NOT EXISTS goim.friend_request (
    `id` bigint not null auto_increment,
    `uid` varchar(64) not null, -- 22 bytes of uuid
    `friend_uid` varchar(64) not null, -- 22 bytes of uuid
    `status` tinyint not null default 0 COMMENT '0: pending; 1: accepted; 2: rejected',
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`uid`, `friend_uid`) COMMENT 'unique key for uid and friend_uid'
) auto_increment = 10000 engine = innodb charset = utf8mb4;

-- define session table based on go structure Session in current directory
DROP TABLE IF EXISTS goim.session;

CREATE TABLE IF NOT EXISTS goim.session (
    `id` bigint not null auto_increment,
    `from_uid` varchar(64) not null, -- 22 bytes of uuid
    `to_uid` varchar(64) not null, -- 22 bytes of uuid
    `status` tinyint not null default 0 COMMENT '0: normal; 1: deleted;',
    `type` tinyint not null default 0 COMMENT '0: single chat; 1: group chat;',
    `created_by` varchar(64) not null, -- 22 bytes of uuid
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`from_uid`, `to_uid`) COMMENT 'unique key for from_uid and to_uid'
) auto_increment = 10000 engine = innodb charset = utf8mb4;

-- define group table based on go structure Group in current directory
DROP TABLE IF EXISTS goim.group;

CREATE TABLE IF NOT EXISTS goim.group (
    `id` bigint not null auto_increment,
    `gid` varchar(64) not null, -- g_ + 22 bytes of uuid
    `name` varchar(64) not null, -- group name
    `description` varchar(255) not null, -- group description
    `avatar` varchar(255) not null, -- group avatar
    `owner_uid` varchar(64) not null, -- 22 bytes of uuid
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`gid`) COMMENT 'unique key for gid'
) auto_increment = 10000 engine = innodb charset = utf8mb4;

-- define group member table based on go structure GroupMember in current directory
DROP TABLE IF EXISTS goim.group_member;

CREATE TABLE IF NOT EXISTS goim.group_member (
    `id` bigint not null auto_increment,
    `gid` varchar(64) not null, -- g_ + 22 bytes of uuid
    `uid` varchar(64) not null, -- 22 bytes of uuid
    `type` tinyint not null default 0 COMMENT '0: owner; 1: member',
    `status` tinyint not null default 0 COMMENT '0: normal; 1: silent;',
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`gid`, `uid`) COMMENT 'unique key for gid and uid'
) auto_increment = 10000 engine = innodb charset = utf8mb4;