all: css js
css: static/gollery.css
js: static/gollery.js

static/gollery.css: assets/less/ assets/less/bootstrap/ assets/less/gollery/
	@echo -n Compiling and minifying CSS...
	@lessc assets/less/bootstrap.less | cssmin > $@
	@echo \ done!

static/gollery.js: assets/js/
	@echo -n Compiling and minifying JS...
	@uglifyjs assets/js/* -m -c -o $@
	@echo \ done!

clean:
	rm -f static/gollery.css static/gollery.js
