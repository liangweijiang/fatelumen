-- MySQL dump 10.13  Distrib 8.0.46, for Linux (x86_64)
--
-- Host: localhost    Database: fatelumen
-- ------------------------------------------------------
-- Server version	8.0.46

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Dumping data for table `annual_calendar_years`
--

LOCK TABLES `annual_calendar_years` WRITE;
/*!40000 ALTER TABLE `annual_calendar_years` DISABLE KEYS */;
INSERT  IGNORE INTO `annual_calendar_years` (`year`, `gan_zhi`, `stem`, `branch`, `stem_element`, `branch_element`, `stem_yin_yang`, `branch_yin_yang`, `zodiac`, `cycle_index`, `data_version`, `created_at`) VALUES (2026,'丙午','丙','午','火','火','阳','阳','马',43,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2027,'丁未','丁','未','火','土','阴','阴','羊',44,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2028,'戊申','戊','申','土','金','阳','阳','猴',45,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2029,'己酉','己','酉','土','金','阴','阴','鸡',46,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2030,'庚戌','庚','戌','金','土','阳','阳','狗',47,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2031,'辛亥','辛','亥','金','水','阴','阴','猪',48,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2032,'壬子','壬','子','水','水','阳','阳','鼠',49,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2033,'癸丑','癸','丑','水','土','阴','阴','牛',50,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2034,'甲寅','甲','寅','木','木','阳','阳','虎',51,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2035,'乙卯','乙','卯','木','木','阴','阴','兔',52,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2036,'丙辰','丙','辰','火','土','阳','阳','龙',53,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2037,'丁巳','丁','巳','火','火','阴','阴','蛇',54,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2038,'戊午','戊','午','土','火','阳','阳','马',55,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2039,'己未','己','未','土','土','阴','阴','羊',56,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2040,'庚申','庚','申','金','金','阳','阳','猴',57,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2041,'辛酉','辛','酉','金','金','阴','阴','鸡',58,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2042,'壬戌','壬','戌','水','土','阳','阳','狗',59,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2043,'癸亥','癸','亥','水','水','阴','阴','猪',60,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2044,'甲子','甲','子','木','水','阳','阳','鼠',1,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2045,'乙丑','乙','丑','木','土','阴','阴','牛',2,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2046,'丙寅','丙','寅','火','木','阳','阳','虎',3,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2047,'丁卯','丁','卯','火','木','阴','阴','兔',4,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2048,'戊辰','戊','辰','土','土','阳','阳','龙',5,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2049,'己巳','己','巳','土','火','阴','阴','蛇',6,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2050,'庚午','庚','午','金','火','阳','阳','马',7,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2051,'辛未','辛','未','金','土','阴','阴','羊',8,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2052,'壬申','壬','申','水','金','阳','阳','猴',9,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2053,'癸酉','癸','酉','水','金','阴','阴','鸡',10,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2054,'甲戌','甲','戌','木','土','阳','阳','狗',11,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2055,'乙亥','乙','亥','木','水','阴','阴','猪',12,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2056,'丙子','丙','子','火','水','阳','阳','鼠',13,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2057,'丁丑','丁','丑','火','土','阴','阴','牛',14,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2058,'戊寅','戊','寅','土','木','阳','阳','虎',15,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2059,'己卯','己','卯','土','木','阴','阴','兔',16,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2060,'庚辰','庚','辰','金','土','阳','阳','龙',17,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2061,'辛巳','辛','巳','金','火','阴','阴','蛇',18,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2062,'壬午','壬','午','水','火','阳','阳','马',19,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2063,'癸未','癸','未','水','土','阴','阴','羊',20,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2064,'甲申','甲','申','木','金','阳','阳','猴',21,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2065,'乙酉','乙','酉','木','金','阴','阴','鸡',22,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2066,'丙戌','丙','戌','火','土','阳','阳','狗',23,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2067,'丁亥','丁','亥','火','水','阴','阴','猪',24,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2068,'戊子','戊','子','土','水','阳','阳','鼠',25,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2069,'己丑','己','丑','土','土','阴','阴','牛',26,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2070,'庚寅','庚','寅','金','木','阳','阳','虎',27,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2071,'辛卯','辛','卯','金','木','阴','阴','兔',28,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2072,'壬辰','壬','辰','水','土','阳','阳','龙',29,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2073,'癸巳','癸','巳','水','火','阴','阴','蛇',30,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2074,'甲午','甲','午','木','火','阳','阳','马',31,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2075,'乙未','乙','未','木','土','阴','阴','羊',32,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2076,'丙申','丙','申','火','金','阳','阳','猴',33,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2077,'丁酉','丁','酉','火','金','阴','阴','鸡',34,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2078,'戊戌','戊','戌','土','土','阳','阳','狗',35,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2079,'己亥','己','亥','土','水','阴','阴','猪',36,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2080,'庚子','庚','子','金','水','阳','阳','鼠',37,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2081,'辛丑','辛','丑','金','土','阴','阴','牛',38,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2082,'壬寅','壬','寅','水','木','阳','阳','虎',39,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2083,'癸卯','癸','卯','水','木','阴','阴','兔',40,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2084,'甲辰','甲','辰','木','土','阳','阳','龙',41,'sexagenary-calendar-v1','2026-08-30 12:01:22.286'),(2085,'乙巳','乙','巳','木','火','阴','阴','蛇',42,'sexagenary-calendar-v1','2026-08-30 12:01:22.286');
/*!40000 ALTER TABLE `annual_calendar_years` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-08-30 12:26:12
