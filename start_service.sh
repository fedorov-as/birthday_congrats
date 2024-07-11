cd birthday_congrats/databases
docker-compose up -d

cd ..
go run ./... -mod=vendor

cd databases
docker-compose stop