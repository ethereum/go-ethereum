# Conversion Tool

To make the migration to **node-addon-api** easier, we have provided a script to
help complete some tasks.

## To use the conversion script:

  1. Go to your module directory

```
cd [module_path]
```

  2. Install node-addon-api module

```
npm install node-addon-api
```
  3. Run node-addon-api conversion script

```
node ./node_modules/node-addon-api/tools/conversion.js ./
```

  4. While this script makes conversion easier, it still cannot fully convert
the module. The next step is to try to build the module and complete the
remaining conversions necessary to allow it to compile and pass all of the
module's tests.