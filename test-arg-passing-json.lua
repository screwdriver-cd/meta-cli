local json = require('json')

meta.set('myjson', json.decode(arg[1]))

assert(meta.get('myjson.foo') == "bar", meta.get('myjson.foo'))
assert(meta.get('myjson.bar[0]') == 1)
assert(meta.get('myjson.bar[1]') == 2)
assert(meta.get('myjson.bar[2]') == 3.45)
