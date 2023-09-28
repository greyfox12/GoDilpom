create table if not exists user_ref (
	user_id  integer NOT NULL GENERATED ALWAYS AS identity,
	login 	 varchar(50) not null,
	user_pass varchar(50) not null,
	ballans	  numeric  default 0,
	withdrawn numeric default 0,
	PRIMARY KEY (user_id),
	UNIQUE (login)
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

-- Function
---------------------------------
-- User registred
create or replace function register( p_login varchar, p_password varchar ) returns integer as $$
 declare ret integer;
 BEGIN
	INSERT INTO user_ref (login, user_pass)
    VALUES (p_login,  p_password)
    ON CONFLICT (login) DO NOTHING 
    returning user_id into ret;
   
   if ret  is null then return 409;
   	          else return 200;
              end if;
  end;
   	$$ LANGUAGE plpgsql;
end;

-- User loging
create or replace function loging( p_login varchar, p_password varchar ) returns integer as $$
 declare ret integer;
 BEGIN
	select count(*) into ret from  user_ref 
        where  login = p_login and user_pass = p_password;
   
   if ret > 0 then return 200;
   	          else return 401;
   end if;
  end;
   	$$ LANGUAGE plpgsql;
end;

-- User LoadOrder 
create or replace function load_order( p_login varchar, ordernum varchar ) returns integer as $$
 declare ret integer;
		p_user_id integer;
 BEGIN
	select user_id into p_user_id from  user_ref 
        where  login = p_login;
    
    if not found then return 401; end if;
   
   select o.user_id  into ret from orders o 
     where o.order_number = ordernum;
   
   if found and ret = p_user_id then return 200; end if; 
   if found and ret != p_user_id then return 409; end if; 
    
   insert into orders(user_id, order_number, order_status) VALUES(p_user_id, ordernum,'NEW');
   
   return 202;
  end;
   	$$ LANGUAGE plpgsql;
end;

-- For trigger WITHDRAW 
CREATE OR REPLACE FUNCTION get_order()
 returns varchar(50) as $$
 declare ret varchar(50);
	     p_id INTEGER;
 begin
   select o.order_number, id  into ret, p_id  from orders o 
    where order_status = 'NEW' or 
         (order_status = 'PROCESSING' and trunc(EXTRACT(
            EPOCH from now() -o.update_at)) > 120 )
   order by update_at 
   limit 1
   for update nowait;
   
   if not found then return null; end if; 
  
   update orders o 
     set order_status = 'PROCESSING',
         update_at  = now()
     where o.id = p_id;
  
  return ret;
end;
$$ language plpgsql;

CREATE OR REPLACE FUNCTION debeting(p_login character varying, ordernum character varying, summ numeric)
 RETURNS integer
 LANGUAGE plpgsql
AS $function$
 declare ret integer;
		p_user_id integer;
 begin
	select user_id, withdrawn  into p_user_id, ret from  user_ref 
        where  login = p_login;
    
    if not found then return 401; end if;
    if ret < summ then return 402; end if;
   
    insert into withdraw (user_id, order_number, summa) VALUES(p_user_id, ordernum, summ);

   update user_ref u 
      set withdrawn = withdrawn - summ
      where u.user_id = p_user_id; 
   
   return 200;
  end;
   	$function$;

CREATE OR REPLACE FUNCTION add_accrual(ordernum varchar, status varchar, summ numeric)
 RETURNS integer
 LANGUAGE plpgsql
AS $function$
 declare ret integer;
		p_user_id integer;
 begin
	select o.user_id into p_user_id from orders o 
	    where o.order_number = ordernum;
	if not found then return 500; end if;
    
    update orders 
      set order_status = status,
          accrual  = summ,
          update_at = now()
    where order_number = ordernum;
   
    update user_ref 
    set ballans  = ballans  + summa
    where user_id = p_user_id;
   
   return 0;
  end;
   	$function$;
   

