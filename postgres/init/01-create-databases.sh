#!/bin/bash

set -e

echo "Creating application databases and users..."

psql \
    --username "$POSTGRES_USER" \
    --dbname "$POSTGRES_DB" \
    --set ON_ERROR_STOP=1 \
    --set user_db_name="$USER_DB_NAME" \
    --set user_db_user="$USER_DB_USER" \
    --set user_db_password="$USER_DB_PASSWORD" \
    --set booking_db_name="$BOOKING_DB_NAME" \
    --set booking_db_user="$BOOKING_DB_USER" \
    --set booking_db_password="$BOOKING_DB_PASSWORD" \
    --set notification_db_name="$NOTIFICATION_DB_NAME" \
    --set notification_db_user="$NOTIFICATION_DB_USER" \
    --set notification_db_password="$NOTIFICATION_DB_PASSWORD" <<'SQL'

CREATE ROLE :"user_db_user" LOGIN PASSWORD :'user_db_password';
CREATE DATABASE :"user_db_name" OWNER :"user_db_user";

CREATE ROLE :"booking_db_user" LOGIN PASSWORD :'booking_db_password';
CREATE DATABASE :"booking_db_name" OWNER :"booking_db_user";

CREATE ROLE :"notification_db_user" LOGIN PASSWORD :'notification_db_password';
CREATE DATABASE :"notification_db_name" OWNER :"notification_db_user";

SQL

echo "Application databases and users created successfully."