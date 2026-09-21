CREATE TABLE IF NOT EXISTS client_tags(
    client_id       UUID, 
    tag_id          UUID,
    FOREIGN KEY (client_id) REFERENCES clients(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id) 
);