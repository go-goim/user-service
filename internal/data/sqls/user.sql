-- drop database
DROP DATABASE IF EXISTS `goim`;

-- create database
create database if not exists goim;

-- define user table based on go structure User in current directory
DROP TABLE IF EXISTS goim.user;

create table if not exists goim.user (
	`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	`uid` BIGINT not null,
	`name` varchar(32) not null,
	`password` varchar(128) not null,
	`email` varchar(64),
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

-- define friend table based on go structure Friend in current directory
DROP TABLE IF EXISTS goim.friend;

CREATE TABLE IF NOT EXISTS goim.friend (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid` BIGINT not null,
    `friend_uid` BIGINT not null,
    `status` tinyint not null default 0 COMMENT '0: friend; 1: stranger; 2: blacked',
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`uid`, `friend_uid`) COMMENT 'uid and friend_uid are unique'
) auto_increment = 10000 engine = innodb charset = utf8mb4;

-- define friend_request table based on go structure FriendRequest in current directory
DROP TABLE IF EXISTS goim.friend_request;

CREATE TABLE IF NOT EXISTS goim.friend_request (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid` BIGINT not null,
    `friend_uid` BIGINT not null,
    `status` tinyint not null default 0 COMMENT '0: pending; 1: accepted; 2: rejected',
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`uid`, `friend_uid`) COMMENT 'unique key for uid and friend_uid'
) auto_increment = 10000 engine = innodb charset = utf8mb4;

-- define group table based on go structure Group in current directory
DROP TABLE IF EXISTS goim.group;

CREATE TABLE IF NOT EXISTS goim.group (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `gid` BIGINT not null,
    `name` varchar(64) not null, -- group name
    `description` varchar(255) not null, -- group description
    `avatar` varchar(255) not null, -- group avatar
    `owner_uid` varchar(64) not null, -- 22 bytes of uuid
    `max_members` int not null default 0, -- max members in group
    `member_count` int not null default 0, -- current members in group
    `status` tinyint not null default 0 COMMENT '0: normal; 1: silent;',
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`gid`) COMMENT 'unique key for gid'
) auto_increment = 10000 engine = innodb charset = utf8mb4;

-- define group member table based on go structure GroupMember in current directory
DROP TABLE IF EXISTS goim.group_member;

CREATE TABLE IF NOT EXISTS goim.group_member (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `gid` BIGINT not null,
    `uid` BIGINT not null,
    `type` tinyint not null default 0 COMMENT '0: owner; 1: member',
    `status` tinyint not null default 0 COMMENT '0: normal; 1: silent;',
    `created_at` int not null default 0,
    `updated_at` int not null default 0,
    primary key (`id`),
    unique key (`gid`, `uid`) COMMENT 'unique key for gid and uid'
) auto_increment = 10000 engine = innodb charset = utf8mb4;
