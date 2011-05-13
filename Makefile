SERVER_BINARY=chunkymonkey
EXTRA_BINARIES=intercept inspectlevel datatests
DIAGRAMS=diagrams/top-level-architecture.png

server: $(SERVER_BINARY)

all: server extra

extra: $(EXTRA_BINARIES)

# godag leaves garbage behind.
cleantmp:
	@-rm -rf src/tmp*

clean:
	@-rm -f $(SERVER_BINARY) $(EXTRA_BINARIES)
	@-rm -rf test_obj
	@gd -q -c

fmt:
	@gd -q -fmt --tab src

test: cleantmp datatests
	@-rm -rf test_obj/tmp*
	@mkdir -p test_obj
	@gd -q -L test_obj -t src
	@./datatests

# Note that this will also compile code in the src/util directory.
libs: cleantmp
	@gd -q src

chunkymonkey: libs
	@gd -q -I src -o $@ src/main

intercept: libs
	@gd -q -I src -o $@ src/util/$@

inspectlevel: libs
	@gd -q -I src -o $@ src/util/$@

datatests: libs
	@gd -q -I src -o $@ src/util/$@

docs: $(DIAGRAMS)

%.png: %.dot
	@dot -Tpng $< -o $@

.PHONY: all clean docs extra fmt libs server test
