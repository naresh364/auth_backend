-- MySQL Workbench Forward Engineering

SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION';

-- -----------------------------------------------------
-- Schema database_name_
-- -----------------------------------------------------
DROP SCHEMA IF EXISTS `database_name_` ;

-- -----------------------------------------------------
-- Schema database_name_
-- -----------------------------------------------------
CREATE SCHEMA IF NOT EXISTS `database_name_` ;
USE `database_name_` ;

-- -----------------------------------------------------
-- Table `database_name_`.`user_role`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `database_name_`.`user_role` ;

CREATE TABLE IF NOT EXISTS `database_name_`.`user_role` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `role` VARCHAR(16) NOT NULL,
  `desc` VARCHAR(511) NULL,
  `date_add` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  `date_upd` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `role_UNIQUE` (`role` ASC) VISIBLE)
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `database_name_`.`org`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `database_name_`.`org` ;

CREATE TABLE IF NOT EXISTS `database_name_`.`org` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(63) NOT NULL,
  `date_add` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  `date_upd` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `database_name_`.`auth_user`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `database_name_`.`auth_user` ;

CREATE TABLE IF NOT EXISTS `database_name_`.`auth_user` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(16) NULL,
  `email` VARCHAR(255) NULL,
  `password` VARCHAR(255) NULL,
  `user_role_id` INT NOT NULL,
  `is_active` TINYINT(1) NULL DEFAULT 0,
  `org_id` INT NOT NULL,
  `facebook_id` VARCHAR(128) NULL,
  `google_id` VARCHAR(128) NULL,
  `date_add` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  `date_upd` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `id_UNIQUE` (`id` ASC) VISIBLE,
  INDEX `fk_auth_user_user_roles1_idx` (`user_role_id` ASC) VISIBLE,
  INDEX `idx_username` (`username` ASC) VISIBLE,
  INDEX `idx_is_active` (`is_active` ASC) VISIBLE,
  INDEX `fk_auth_user_org1_idx` (`org_id` ASC) VISIBLE,
  CONSTRAINT `fk_auth_user_user_roles1`
    FOREIGN KEY (`user_role_id`)
    REFERENCES `database_name_`.`user_role` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION,
  CONSTRAINT `fk_auth_user_org1`
    FOREIGN KEY (`org_id`)
    REFERENCES `database_name_`.`org` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION);


-- -----------------------------------------------------
-- Table `database_name_`.`user_role_permission`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `database_name_`.`user_role_permission` ;

CREATE TABLE IF NOT EXISTS `database_name_`.`user_role_permission` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `table_name` VARCHAR(45) NOT NULL,
  `column_name` VARCHAR(45) NOT NULL DEFAULT '*',
  `value` VARCHAR(255) NULL,
  `permission` ENUM("r", "u", "c", "d", "*") NOT NULL DEFAULT 'r',
  `user_role_id` INT NOT NULL,
  `date_add` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  `date_upd` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `fk_user_permission_user_roles_idx` (`user_role_id` ASC) VISIBLE,
  CONSTRAINT `fk_user_permission_user_roles`
    FOREIGN KEY (`user_role_id`)
    REFERENCES `database_name_`.`user_role` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;

-- -----------------------------------------------------
-- Table `database_name_`.`user_permission`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `database_name_`.`user_permission` ;

CREATE TABLE IF NOT EXISTS `database_name_`.`user_permission` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `auth_user_id` INT NOT NULL,
  `table_name` VARCHAR(45) NOT NULL,
  `column_name` VARCHAR(45) NOT NULL,
  `value` VARCHAR(255) NULL,
  `permission` ENUM("r", "u", "c", "d", "*") NOT NULL DEFAULT 'r',
  `org_id` INT NOT NULL,
  `date_add` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  `date_upd` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `fk_user_permission_auth_user1_idx` (`auth_user_id` ASC) VISIBLE,
  INDEX `fk_user_permission_org1_idx` (`org_id` ASC) VISIBLE,
  CONSTRAINT `fk_user_permission_auth_user1`
    FOREIGN KEY (`auth_user_id`)
    REFERENCES `database_name_`.`auth_user` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION,
  CONSTRAINT `fk_user_permission_org1`
    FOREIGN KEY (`org_id`)
    REFERENCES `database_name_`.`org` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;

DROP TABLE IF EXISTS `database_name_`.`test_table` ;

CREATE TABLE IF NOT EXISTS `database_name_`.`test_table` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(127) NOT NULL,
  `s_value` ENUM('A', 'B', 'C') NULL DEFAULT 'A',
  `i_value` INT NULL,
  `u_value` INT UNSIGNED NULL,
  `f_value` FLOAT NULL,
  `d_value` FLOAT NULL,
  `auth_user_id` INT NOT NULL,
  `org_id` INT NOT NULL,
  `date_add` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  `date_upd` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `fk_test_table_auth_user1_idx` (`auth_user_id` ASC) VISIBLE,
  INDEX `idx_name` (`name` ASC) VISIBLE,
  INDEX `fk_test_table_org1_idx` (`org_id` ASC) VISIBLE,
  CONSTRAINT `fk_test_table_auth_user1`
    FOREIGN KEY (`auth_user_id`)
    REFERENCES `database_name_`.`auth_user` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION,
  CONSTRAINT `fk_test_table_org1`
    FOREIGN KEY (`org_id`)
    REFERENCES `database_name_`.`org` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;


CREATE TABLE IF NOT EXISTS `database_name_`.`test_table2` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(127) NOT NULL,
  `i_value` INT NULL,
  `auth_user_id` INT NOT NULL,
  `org_id` INT NOT NULL,
  `test_table_id` INT NULL,
  `date_add` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  `date_upd` TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `fk_test_table2_auth_user1_idx` (`auth_user_id` ASC) VISIBLE,
  INDEX `idx_name` (`name` ASC) VISIBLE,
  INDEX `fk_test_table2_org1_idx` (`org_id` ASC) VISIBLE,
  INDEX `fk_test_table2_test_table21_idx` (`test_table_id` ASC) VISIBLE,
  CONSTRAINT `fk_test_table2_auth_user1`
    FOREIGN KEY (`auth_user_id`)
    REFERENCES `database_name_`.`auth_user` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION,
  CONSTRAINT `fk_test_table_org2`
    FOREIGN KEY (`org_id`)
    REFERENCES `database_name_`.`org` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION,
  CONSTRAINT `fk_test_table_test_table2`
    FOREIGN KEY (`test_table_id`)
    REFERENCES `database_name_`.`test_table` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;




insert into user_role set id=1, role="super_admin";
insert into user_role set id=2, role="another_user";
insert into org set id=1, name="super_users";
insert into org set id=2, name="another_org";

insert into user_role_permission set table_name='test_table2', column_name='', permission='r', user_role_id=2;
insert into user_role_permission set table_name='test_table2', column_name='', permission='c', user_role_id=2;
insert into user_role_permission set table_name='test_table2', column_name='', permission='d', user_role_id=2;
insert into user_role_permission set table_name='test_table2', column_name='', permission='u', user_role_id=2;

insert into user_role_permission set table_name='test_table', column_name='s_value', value='B', permission='r', user_role_id=2;
insert into user_role_permission set table_name='test_table', column_name='s_value', value='C', permission='r', user_role_id=2;
insert into user_role_permission set table_name='test_table', column_name='s_value', value='A', permission='u', user_role_id=2;
insert into user_role_permission set table_name='test_table', column_name='f_value', value='1.1', permission='c', user_role_id=2;
insert into user_role_permission set table_name='test_table', column_name='f_value', value='1.2', permission='c', user_role_id=2;
insert into user_role_permission set table_name='test_table', column_name='d_value', value='2.1', permission='c', user_role_id=2;
insert into user_role_permission set table_name='test_table', column_name='i_value', value='-1', permission='d', user_role_id=2;
insert into user_role_permission set table_name='test_table', column_name='u_value', value='1', permission='d', user_role_id=2;

-- -----------------------------------------------------
-- Password  : nkktest
-- -----------------------------------------------------
insert into auth_user set id=1, username="su", user_role_id=1, org_id=1, is_active=1, password="$2a$04$p9zm7fqZVajMyiSE1bXgl.kJpt4Nw2mOzdAoY57Wp43NqVJ.kGMOq";
insert into auth_user set id=2, username="simple_user", user_role_id=2, org_id=2, is_active=1, password="$2a$04$p9zm7fqZVajMyiSE1bXgl.kJpt4Nw2mOzdAoY57Wp43NqVJ.kGMOq";

insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("yes", 'B', 1, 1, 1.1, 1.2, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("yes", 'B', 1, 2, 1.1, 1.2, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("yes", 'C', 1, 3, 1.1, 1.2, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("yes", 'C', 1, 4, 1.1, 1.2, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("yes", 'C', 1, 5, 1.1, 1.2, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("no", 'B', 1, 1, 1.1, 1.2, 1, 1);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("no", 'B', 1, 1, 1.1, 1.2, 2, 1);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("no", 'B', 1, 1, 1.1, 1.3, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("no", 'B', 1, 1, 1.4, 1.2, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("no", 'B', 1, 7, 1.1, 1.2, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("no", 'B', 2, 1, 1.1, 1.2, 2, 2);
insert into test_table(name, s_value, i_value, u_value, f_value, d_value, auth_user_id, org_id) values("no", 'A', 1, 1, 1.1, 1.2, 2, 2);

use `database_name_`;