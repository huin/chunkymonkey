BINARIES=\
	bin/chunkymonkey \
	bin/datatests \
	bin/inspectlevel \
	bin/intercept \
	bin/noise \
	bin/replay \
	bin/style

MOCK_FILES=\
	src/chunkymonkey/gamerules/mock_stub_test.go \
	src/chunkymonkey/physics/mock_physics_test.go

GD_OPTS=-quiet

DIAGRAMS=\
	diagrams/top-level-architecture.png \
	diagrams/deps.png

all: $(BINARIES)

clean:
	@-rm -f $(BINARIES)
	@-rm -f $(MOCK_FILES)
	@gd $(GD_OPTS) -lib _obj -clean src
	@gd $(GD_OPTS) -lib _test -clean .

fmt:
	@gd $(GD_OPTS) -fmt -tab src

check: bin/style
	@bin/style `find . -name \*.go`

test: mocks
	@gd $(GD_OPTS) -lib _test -test src

bench: mocks
	@gd $(GD_OPTS) -lib _test -bench 'Benchmark' -match '^$$' -test src

libs:
	@gd $(GD_OPTS) -lib _obj src

test_data: bin/datatests
	@bin/datatests

bin/%: libs
	@gd $(GD_OPTS) -I _obj -lib _obj -M cmd/$*/main -output $@ src

docs: $(DIAGRAMS)

%.png: %.dot
	@dot -Tpng $< -o $@

diagrams/deps.dot:
	@gd -dot $@ src
	@# Omit dependencies upon packages not in the project for clarity, and
	@# omit chunkymonkey/types which essentially everything depends on.
	@sed -ri '/->/{/"[^"]+" -> "(cmd|chunkymonkey|nbt|perlin|testencoding|testmatcher)[/"]/b ok ; d ; : ok} ; /chunkymonkey\/types/d' $@

mocks: $(MOCK_FILES)

src/chunkymonkey/gamerules/mock_stub_test.go: src/chunkymonkey/gamerules/stub.go
	mockgen -package gamerules -destination $@ -source $< -imports .=chunkymonkey/types

src/chunkymonkey/physics/mock_physics_test.go: src/chunkymonkey/physics/physics.go
	mockgen -package physics -destination $@ -source $< -imports .=chunkymonkey/types


.PHONY: all bench check clean docs fmt mocks test test_data
