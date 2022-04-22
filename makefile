VERSION=latest
PLATFORM=local

docker: docker-poly docker-mono

docker-mono:
	@docker build . --target image-monolith --platform ${PLATFORM} -t 192.168.1.15:15001/dendrite-monolith:${VERSION}
	#@docker push 192.168.1.15:15001/dendrite-monolith:${VERSION}

docker-poly:
	@docker build . --target image-polylith --platform ${PLATFORM} -t 192.168.1.15:15001/dendrite-polylith:${VERSION}
	@docker push 192.168.1.15:15001/dendrite-polylith:${VERSION}

docker-complement:
	#@docker build . --target image-complement --platform ${PLATFORM} -t complement-dendrite
	@docker build . -t complement-dendrite -f build/scripts/ComplementLocal.Dockerfile

###

srcDir := $(shell pwd)

sytest-postgres:
	POSTGRES=1 ./scripts/sytest.sh ${srcDir}
	cat postgres.duration

sytest-postgres-api:
	POSTGRES=1 API=1 ./scripts/sytest.sh ${srcDir}
	cat postgres_full_api.duration

sytest-sqlite:
	./scripts/sytest.sh ${srcDir}
	cat sqlite.duration

sytest-sqlite-api:
	API=1 ./scripts/sytest.sh ${srcDir}
	cat sqlite_full_api.duration

sytest: clean sytest-sqlite sytest-sqlite-api sytest-postgres sytest-postgres-api

benchmark: sytest
	tail -n1 *.duration

clean:
	sudo rm -rf sytest_*
	rm -rf *.db *.duration *.log
