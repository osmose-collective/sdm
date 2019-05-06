.PHONY: docker.up
docker.up:
	docker-compose build --pull
	docker-compose up -d --force-recreate --remove-orphans
