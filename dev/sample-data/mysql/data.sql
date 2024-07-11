CREATE TABLE user (
    user_id INT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL
);

CREATE TABLE profile (
    profile_id INT PRIMARY KEY,
    user_id INT,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    bio TEXT,
    FOREIGN KEY (user_id) REFERENCES user(user_id)
);

INSERT INTO user (user_id, username, email) VALUES
(1, 'john_doe', 'john_doe@example.com'),
(2, 'jane_smith', 'jane_smith@example.com');

INSERT INTO profile (profile_id, user_id, first_name, last_name, bio) VALUES
(1, 1, 'John', 'Doe', 'Software Developer at XYZ Corp.'),
(2, 2, 'Jane', 'Smith', 'Data Scientist at ABC Inc.');
