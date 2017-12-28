# DESCRIPTION
# Retrieves and situates dependences needed to run / compile all the niffly-willers for https://github.com/tehmoon/cryptocli

# USAGE
# Navigate to the go/src directory of your Go install. Copy this script there. Run this script:
# ./installDependencies.sh


# CODE
if [ ! -d github.com]
then
	mkdir github.com
fi
exit

cd github.com

mkdir aws
cd aws
git clone https://github.com/aws/aws-sdk-go.git
cd ..

mkdir pkg
cd pkg
git clone https://github.com/pkg/errors.git
cd ..

mkdir tehmoon
cd tehmoon
git clone https://github.com/tehmoon/errors.git

cd ..

go get golang.org/x/crypto/blake2b