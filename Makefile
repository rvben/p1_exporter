run:
	go run main.go

build:
	go build .

build_arm:
	GOOS=linux GOARCH=arm GOARM=7 go build

deploy:
	ansible-playbook -i inventory.ini main.yml

