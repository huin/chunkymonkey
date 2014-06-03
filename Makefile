MOCK_FILES=\
	gamerules/mock_stub_test.go \
	physics/mock_physics_test.go

mocks: $(MOCK_FILES)

clean:
	rm $(MOCK_FILES)

gamerules/mock_stub_test.go: gamerules/stub.go
	mockgen -package gamerules -destination $@ -source $< -imports .=github.com/huin/chunkymonkey/types

physics/mock_physics_test.go: physics/physics.go
	mockgen -package physics -destination $@ -source $< -imports .=github.com/huin/chunkymonkey/types


.PHONY: mocks
