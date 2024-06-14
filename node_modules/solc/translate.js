var linker = require('./linker.js');

/// Translate old style version numbers to semver.
/// Old style: 0.3.6-3fc68da5/Release-Emscripten/clang
///            0.3.5-371690f0/Release-Emscripten/clang/Interpreter
///            0.3.5-0/Release-Emscripten/clang/Interpreter
///            0.2.0-e7098958/.-Emscripten/clang/int linked to libethereum-1.1.1-bbb80ab0/.-Emscripten/clang/int
///            0.1.3-0/.-/clang/int linked to libethereum-0.9.92-0/.-/clang/int
///            0.1.2-5c3bfd4b*/.-/clang/int
///            0.1.1-6ff4cd6b/RelWithDebInfo-Emscripten/clang/int
/// New style: 0.4.5+commit.b318366e.Emscripten.clang
function versionToSemver (version) {
  // FIXME: parse more detail, but this is a good start
  var parsed = version.match(/^([0-9]+\.[0-9]+\.[0-9]+)-([0-9a-f]{8})[/*].*$/);
  if (parsed) {
    return parsed[1] + '+commit.' + parsed[2];
  }
  if (version.indexOf('0.1.3-0') !== -1) {
    return '0.1.3';
  }
  if (version.indexOf('0.3.5-0') !== -1) {
    return '0.3.5';
  }
  // assume it is already semver compatible
  return version;
}

function translateErrors (ret, errors) {
  for (var error in errors) {
    var type = 'error';
    var extractType = /^(.*):(\d+):(\d+):(.*):/;
    extractType = extractType.exec(errors[error]);
    if (extractType) {
      type = extractType[4].trim();
    } else if (errors[error].indexOf(': Warning:')) {
      type = 'Warning';
    } else if (errors[error].indexOf(': Error:')) {
      type = 'Error';
    }
    ret.push({
      type: type,
      component: 'general',
      severity: (type === 'Warning') ? 'warning' : 'error',
      message: errors[error],
      formattedMessage: errors[error]
    });
  }
}

function translateGasEstimates (gasEstimates) {
  if (gasEstimates === null) {
    return 'infinite';
  }

  if (typeof gasEstimates === 'number') {
    return gasEstimates.toString();
  }

  var gasEstimatesTranslated = {};
  for (var func in gasEstimates) {
    gasEstimatesTranslated[func] = translateGasEstimates(gasEstimates[func]);
  }
  return gasEstimatesTranslated;
}

function translateJsonCompilerOutput (output, libraries) {
  var ret = {};

  ret['errors'] = [];
  var errors;
  if (output['error']) {
    errors = [ output['error'] ];
  } else {
    errors = output['errors'];
  }
  translateErrors(ret['errors'], errors);

  ret['contracts'] = {};
  for (var contract in output['contracts']) {
    // Split name first, can be `contract`, `:contract` or `filename:contract`
    var tmp = contract.match(/^(([^:]*):)?([^:]+)$/);
    if (tmp.length !== 4) {
      // Force abort
      return null;
    }
    var fileName = tmp[2];
    if (fileName === undefined) {
      // this is the case of `contract`
      fileName = '';
    }
    var contractName = tmp[3];

    var contractInput = output['contracts'][contract];

    var gasEstimates = contractInput['gasEstimates'];
    var translatedGasEstimates = {};

    if (gasEstimates['creation']) {
      translatedGasEstimates['creation'] = {
        'codeDepositCost': translateGasEstimates(gasEstimates['creation'][1]),
        'executionCost': translateGasEstimates(gasEstimates['creation'][0])
      };
    }
    if (gasEstimates['internal']) {
      translatedGasEstimates['internal'] = translateGasEstimates(gasEstimates['internal']);
    }
    if (gasEstimates['external']) {
      translatedGasEstimates['external'] = translateGasEstimates(gasEstimates['external']);
    }

    var contractOutput = {
      'abi': JSON.parse(contractInput['interface']),
      'metadata': contractInput['metadata'],
      'evm': {
        'legacyAssembly': contractInput['assembly'],
        'bytecode': {
          'object': contractInput['bytecode'] && linker.linkBytecode(contractInput['bytecode'], libraries || {}),
          'opcodes': contractInput['opcodes'],
          'sourceMap': contractInput['srcmap'],
          'linkReferences': contractInput['bytecode'] && linker.findLinkReferences(contractInput['bytecode'])
        },
        'deployedBytecode': {
          'object': contractInput['runtimeBytecode'] && linker.linkBytecode(contractInput['runtimeBytecode'], libraries || {}),
          'sourceMap': contractInput['srcmapRuntime'],
          'linkReferences': contractInput['runtimeBytecode'] && linker.findLinkReferences(contractInput['runtimeBytecode'])
        },
        'methodIdentifiers': contractInput['functionHashes'],
        'gasEstimates': translatedGasEstimates
      }
    };

    if (!ret['contracts'][fileName]) {
      ret['contracts'][fileName] = {};
    }

    ret['contracts'][fileName][contractName] = contractOutput;
  }

  var sourceMap = {};
  for (var sourceId in output['sourceList']) {
    sourceMap[output['sourceList'][sourceId]] = sourceId;
  }

  ret['sources'] = {};
  for (var source in output['sources']) {
    ret['sources'][source] = {
      id: sourceMap[source],
      legacyAST: output['sources'][source].AST
    };
  }

  return ret;
}

function escapeString (text) {
  return text
    .replace(/\n/g, '\\n')
    .replace(/\r/g, '\\r')
    .replace(/\t/g, '\\t');
}

// 'asm' can be an object or a string
function formatAssemblyText (asm, prefix, source) {
  if (typeof asm === 'string' || asm === null || asm === undefined) {
    return prefix + (asm || '') + '\n';
  }
  var text = prefix + '.code\n';
  asm['.code'].forEach(function (item, i) {
    var v = item.value === undefined ? '' : item.value;
    var src = '';
    if (source !== undefined && item.begin !== undefined && item.end !== undefined) {
      src = escapeString(source.slice(item.begin, item.end));
    }
    if (src.length > 30) {
      src = src.slice(0, 30) + '...';
    }
    if (item.name !== 'tag') {
      text += '  ';
    }
    text += prefix + item.name + ' ' + v + '\t\t\t' + src + '\n';
  });
  text += prefix + '.data\n';
  var asmData = asm['.data'] || [];
  for (var i in asmData) {
    var item = asmData[i];
    text += '  ' + prefix + '' + i + ':\n';
    text += formatAssemblyText(item, prefix + '    ', source);
  }
  return text;
}

function prettyPrintLegacyAssemblyJSON (assembly, source) {
  return formatAssemblyText(assembly, '', source);
}

module.exports = {
  versionToSemver: versionToSemver,
  translateJsonCompilerOutput: translateJsonCompilerOutput,
  prettyPrintLegacyAssemblyJSON: prettyPrintLegacyAssemblyJSON
};
