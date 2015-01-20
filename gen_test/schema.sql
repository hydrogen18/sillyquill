create table trucks (
	id serial not null unique,
	created_at timestamp not null,
	updated_at timestamp not null,
	make varchar not null,
	model varchar not null,
	tonnage real not null,
	PRIMARY KEY(make,model)
);

create table cars (
	id bigserial not null unique,
	updated_at timestamp not null,
	make varchar not null,
	model varchar not null,
	passengers smallint not null,
	PRIMARY KEY(make,model)
);

create table incidents (
	id bigserial,
	created_at timestamp not null,
	resolved_at timestamp,
	resolution varchar,
	reported_by varchar,
	resolved_by varchar
);

create table pizza_delivery_guys (
	name varchar,
	gas_mileage double precision not null,
	pizzas_delivered int,
	PRIMARY KEY(name)
); 

create table archive_files (
	id serial unique,
	name varchar not null,
	data bytea not null
);

create table numbers (
	id serial unique,
	value numeric not null
);

create table null_numbers (
	id serial unique,
	value numeric null,
	title varchar not null
);