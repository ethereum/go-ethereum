import { PathReporter } from './PathReporter';
/**
 * @since 1.0.0
 * @deprecated
 */
export var ThrowReporter = {
    report: function (validation) {
        if (validation.isLeft()) {
            throw PathReporter.report(validation).join('\n');
        }
    }
};
