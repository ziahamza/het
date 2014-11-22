var getSearchResults = function(query) {
	return new Promise(function(resolve, reject) {
		var req = new XMLHttpRequest();
		req.open('GET', '/search?query=' + query);
		req.onload = function(res) {
			if (req.status == 200) {
				var obj = null;
				try {
					obj = JSON.parse(req.response);
				}
				catch(ex) { return reject(err) };

				if (!obj.success) {
					return reject(Error(obj.message));
				}

				resolve(obj.results);
			} else {
				reject(Error("Server not responding properly:" + req.status))
			}
		};
		req.onerror = function() {
			reject(Error("Server unavailable."))
		};

		req.send();
	});
};

var SearchApp = React.createClass({
	displayName: 'SearchApp',
	getInitialState: function() {
	console.log('initializing state ...');
		var self = this;
		var textBus = new Bacon.Bus();

		var text = textBus.debounce(300);

		var suggestions = text.flatMapLatest(function(query) {
			if (query.length < 3) {
				return Bacon.once([]);
			}

			console.log('got back query:', query);

			return Bacon.fromPromise(getSearchResults(query));
		});

		text.awaiting(suggestions).onValue(function(x) {
			if (x) {
				self.setState({ loading: true })
			}
		});

		suggestions.onValue(function(results) {
			console.log('sending reesults: ', results)
			self.setState({ loading: false, results: results });
		});

		return {
			welcome: true,
			textBus: textBus,
			query: ''
		}
	},
	handleChange: function(ev) {
		this.state.textBus.push(ev.target.value);

		this.setState({ welcome: false, loading: true, query: ev.target.value });
	},
	handleSubmit: function(ev) {
		ev.preventDefault();

		this.setState({ welcome: false })
	},
	render: function() {
		console.log(this.state);

		var results;

		if (this.state.welcome) {
			results = null;
		}
		else if(this.state.query.length < 3) {
			results = React.createElement('b', null, 'Query too small. Input a bigger search Query.');
		}
		else if (this.state.loading) {
			results = React.createElement('b', null, 'Loading ...');
		}
		else if (this.state.results.length) {
			results = this.state.results.map(function(r) {
				return React.createElement('div', {
					className: 'result panel panel-default'
				},
					React.createElement('div', {
						className: 'panel-heading'
					},
						React.createElement('a', {className: 'link'}, r.Doc.Title),
						React.createElement('small', {className: 'text-muted'}, ' - ' + r.Link.LastModified)
					),

					React.createElement('div', {
						className: 'panel-body'
					}, r.URL)
				);
			});
		}
		else {
			results = React.createElement('b', null, 'No results found.');
		}

		console.log('rendering results:', results);

		return React.createElement('div', {className: 'container'},
			React.createElement('form', {
				className: 'middle ' + (this.state.welcome ? 'searchbox' : 'searchbox-animated'),
				onSubmit: this.handleSubmit
			},
				React.createElement('input', {
					type: 'text',
					onChange: this.handleChange,
					className: 'form-control input-lg searchtext',
					placeholder: 'Search',
					autoComplete: 'off'
				}),
				React.createElement('button', {
					type: 'submit',
					className: 'btn btn-info submitbtn',
					disabled: !!this.state.loading,
				}, React.createElement('span', { className: 'glyphicon glyphicon-search' }))
			),
			React.createElement('div', {className: 'middle'}, results)
		);
	},
});

var searchRoot = React.createElement(SearchApp, null);

React.render(searchRoot, document.querySelector('body'))
