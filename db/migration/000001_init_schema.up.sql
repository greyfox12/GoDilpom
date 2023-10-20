create table if not exists user_ref (
	user_id  integer NOT NULL GENERATED ALWAYS AS identity,
	login 	 varchar(50) not null,
	user_pass varchar(150) not null,
	ballans	  numeric  default 0,
	withdrawn numeric default 0,
	PRIMARY KEY (user_id),
	UNIQUE (login),
	CONSTRAINT user_ref_check CHECK ((ballans >= (0)::numeric))
);

create table if not exists orders (
    id               integer not null GENERATED ALWAYS AS identity,
	user_id          integer NOT null references user_ref(user_id),
	order_number 	 varchar(50) not null,
	order_status     varchar(20) not null,
	uploaded_at      timestamp with time zone default (now() at time zone 'msk'),
	accrual	         numeric default 0,
	update_at		 timestamp with time zone default (now() at time zone 'msk'),
	PRIMARY KEY (id),
	UNIQUE (order_number)
);

create table if not exists withdraw(
    id              integer not null GENERATED ALWAYS AS identity,
	user_id         integer NOT null references user_ref(user_id),
	order_number	varchar(50) not null,
	summa	        numeric not null,
	uploaded_at     timestamp with time zone default (now() at time zone 'msk'),
	PRIMARY KEY (id),
	UNIQUE (order_number)
);
