.PHONY: docker.up
docker.up:
	docker-compose build --pull
	docker-compose up -d --force-recreate --remove-orphans

.PHONY: gen
gen:
	docker-compose build
	docker-compose run -w "$(PWD)" -v "$(PWD):$(PWD)" -e GEN_AND_STOP=1 api
