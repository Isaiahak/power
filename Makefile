venv:
	python3 -m venv venv
	./venv/bin/pip install -r requirements.txt

setup: 
	source venv/bin/activate

run: venv
	./venv/bin/python main.py
