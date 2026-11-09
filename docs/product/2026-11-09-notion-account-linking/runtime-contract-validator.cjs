const fs=require('fs');
const YAML=require(process.env.NOTION_NODE_MODULES + '/yaml');
const Ajv=require(process.env.NOTION_NODE_MODULES + '/ajv/dist/2020').default;
const formats=require(process.env.NOTION_NODE_MODULES + '/ajv-formats');
const spec=YAML.parse(fs.readFileSync(process.env.NOTION_OPENAPI_PATH,'utf8'));
const observed=JSON.parse(fs.readFileSync(0,'utf8'));
let operation=spec.paths[observed.path]?.[observed.method.toLowerCase()];
if(!operation) throw Error('undocumented operation');
let response=operation.responses[String(observed.status)];
if(response?.$ref) response=spec.components.responses[response.$ref.split('/').pop()];
if(!response) throw Error('undocumented status');
if(observed.status===204){if(observed.body!=='')throw Error('unexpected response body');process.exit(0)}
if(observed.status===302){if(!observed.location)throw Error('missing Location');const a=new Ajv({strict:false});formats(a);if(!a.validate({type:'string',format:'uri'},observed.location))throw Error('invalid Location');process.exit(0)}
const type=observed.contentType.split(';')[0];
const contract=response.content?.[type];if(!contract)throw Error('undocumented content type');
let body=type==='application/json'?JSON.parse(observed.body):observed.body.trimEnd();
const ajv=new Ajv({strict:false});formats(ajv);const schema={...contract.schema,components:spec.components};
if(!ajv.validate(schema,body))throw Error('schema validation failed: '+ajv.errorsText());
