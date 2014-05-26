all: static/gollery.css

static/gollery.css: assets/less/ assets/less/bootstrap/ assets/less/gollery/
	@echo -n Compiling and minifying LESS...
	@lessc assets/less/bootstrap.less | cssmin > $@
	@echo \ done!
