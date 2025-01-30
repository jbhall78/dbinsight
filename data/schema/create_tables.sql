-- Simple table for manual inserts
CREATE TABLE products (
    id INT PRIMARY KEY AUTO_INCREMENT,  -- Auto-incrementing ID
    name VARCHAR(255) NOT NULL,      -- Product name
    price DECIMAL(10, 2)              -- Price (10 digits total, 2 decimal places)
);

-- Table with major MySQL datatypes
CREATE TABLE data_types_demo (
    id INT UNSIGNED PRIMARY KEY AUTO_INCREMENT, -- Auto-incrementing unsigned integer ID
    tiny_int_col TINYINT,
    small_int_col SMALLINT,
    medium_int_col MEDIUMINT,
    int_col INT,
    big_int_col BIGINT,
    float_col FLOAT,
    double_col DOUBLE,
    decimal_col DECIMAL(10, 2),
    date_col DATE,
    datetime_col DATETIME,
    timestamp_col TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- Automatically updated
    time_col TIME,
    year_col YEAR,
    char_col CHAR(10),
    varchar_col VARCHAR(255),
    tinytext_col TINYTEXT,
    text_col TEXT,
    mediumtext_col MEDIUMTEXT,
    longtext_col LONGTEXT,
    enum_col ENUM('value1', 'value2', 'value3'),
    set_col SET('option1', 'option2', 'option3'),
    binary_col BINARY(10),
    varbinary_col VARBINARY(255),
    tinyblob_col TINYBLOB,
    blob_col BLOB,
    mediumblob_col MEDIUMBLOB,
    longblob_col LONGBLOB,
    boolean_col BOOLEAN  -- MySQL treats BOOLEAN as TINYINT(1)
);