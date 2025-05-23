/**
 * Checks if its a valid topic
 */
export declare const isTopic: (topic: string) => boolean;
/**
 * Returns true if the topic is part of the given bloom.
 * note: false positives are possible.
 */
export declare const isTopicInBloom: (bloom: string, topic: string) => boolean;
