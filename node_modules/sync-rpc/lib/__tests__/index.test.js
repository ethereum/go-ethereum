const rpc = require('../');

const client = rpc(__dirname + '/../test-worker.js', 'My Server');
test('it should work', () => {
  for (let i = 0; i < 10; i++) {
    expect(client('My Message')).toBe('sent My Message to My Server');
  }
});

let nativeNCFails = false;

rpc.FUNCTION_PRIORITY.forEach((fn, i) => {
  test('profile ' + fn.name, () => {
    try {
      rpc.configuration.fastestFunction = fn;
      const start = Date.now();
      for (let i = 0; i < 100; i++) {
        expect(client('My Message')).toBe('sent My Message to My Server');
      }
      const end = Date.now();
      console.log(fn.name + ': ' + (end - start));
    } catch (ex) {
      if (fn.name === 'nativeNC') {
        console.log(fn.name + ' fails');
        nativeNCFails = true;
        return;
      }
      throw ex;
    }
  });
});

rpc.FUNCTION_PRIORITY.forEach((fn, i) => {
  test('test 30MB ' + fn.name, () => {
    let result;
    try {
      rpc.configuration.fastestFunction = fn;
      result = client('big');
    } catch (ex) {
      if (fn.name === 'nativeNC' && nativeNCFails) {
        console.log(fn.name + ' fails');
        return;
      }
      throw ex;
    }
    expect(result.length).toBe(30 * 1024 * 1024, 42);
    // for (let i = 0; i < 30 * 1024 * 1024, 42; i++) {
    //   expect(result[i]).toBe(42);
    // }
  });
});

rpc.FUNCTION_PRIORITY.forEach((fn, i) => {
  let longMessage = '';
  for (let i = 0; i < 100000; i++) {
    longMessage += 'My Long Message Content';
  }
  test('profile large ' + fn.name, () => {
    try {
      rpc.configuration.fastestFunction = fn;
      const start = Date.now();
      for (let i = 0; i < 10; i++) {
        expect(client(longMessage)).toBe(`sent ${longMessage} to My Server`);
      }
      const end = Date.now();
      console.log('large ' + fn.name + ': ' + (end - start));
    } catch (ex) {
      if (fn.name === 'nativeNC' && nativeNCFails) {
        console.log('large ' + fn.name + ' fails');
        return;
      }
      throw ex;
    }
  });
});
