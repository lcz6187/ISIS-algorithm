MP1_NODE_DIR=mp1_node
RESOURCES_DIR=resources
OUTPUT=./bin

.PHONY: all build_mp1_node run_mp1_node copy_resources clean

all: build_mp1_node copy_resources

build_mp1_node:
	mkdir -p $(OUTPUT)
	go build -C $(MP1_NODE_DIR) -o ../$(OUTPUT)/mp1_node .

# Not useful for now
# run_mp1_node:
# 	go run -C $(MP1_NODE_DIR) .

copy_resources:
	mkdir -p $(OUTPUT)
	cp -r $(RESOURCES_DIR)/* $(OUTPUT)/

clean:
	rm -rf $(OUTPUT)/*
