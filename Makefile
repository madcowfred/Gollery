JS = assets/js/modernizr.custom.js assets/js/grid.js assets/js/gollery.js

all: css js
css: static/gollery.min.css
js: static/gollery.min.js

static/gollery.min.css: assets/less/ assets/less/bootstrap/ assets/less/gollery/
	@echo -n Compiling and minifying CSS...
	@lessc assets/less/bootstrap.less | cssmin > $@
	@echo \ done!

static/gollery.min.js: assets/js/
	@echo -n Compiling and minifying JS...
	@uglifyjs ${JS} -m -c -o $@
	@echo \ done!

clean:
	rm -f static/gollery.min.css static/gollery.min.js
