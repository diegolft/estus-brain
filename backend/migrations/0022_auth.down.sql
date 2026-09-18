-- sessions first: it holds the foreign key into users, so dropping users
-- ahead of it would be refused.
drop table if exists sessions;
drop table if exists users;
