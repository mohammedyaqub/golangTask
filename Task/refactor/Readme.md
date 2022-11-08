## To run the following program need mysql database.
### Create two table's to keep track for login creadentials and orders.

#### Fist create a database. 
```
create database orders;
```
#### 1. Create a register table for login creadentials.

```
use orders;
create table register(
email varchar(30)not null,
password varchar(30) not null,
phone varchar(10),
name varchar(30)
);
```
#### 2. Create a orders table using same database. 
```create database orders;

use orders;
CREATE TABLE ORDERS (
Order_Id int NOT NULL AUTO_INCREMENT,	
CustomerName varchar(255) NOT NULL,  
OrderQuantity int NOT NULL default 0,
PRIMARY KEY (Order_Id)
);
```
