create table trucks (
	id serial not null unique,
	created_at timestamp not null,
	updated_at timestamp not null,
	make varchar not null,
	model varchar not null,
	tonnage real not null
);

create table cars (
	id bigserial not null unique,
	updated_at timestamp not null,
	make varchar not null,
	model varchar not null,
	passengers smallint not null
);