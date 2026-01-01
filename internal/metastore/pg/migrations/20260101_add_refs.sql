-- migrate:up
CREATE TABLE refs (
    repo_name VARCHAR(255) NOT NULL REFERENCES repositories(name) ON DELETE CASCADE,
    ref_name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,
    hash VARCHAR(40),
    target VARCHAR(255),
    PRIMARY KEY (repo_name, ref_name)
);

-- migrate:down
DROP TABLE refs;
