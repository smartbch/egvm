import {Bip32Key} from "../extension/bip32key";

export interface EGVMContext {
    readonly GetConfig: () => string;
    readonly SetConfig: (cfg: string) => void;
    readonly GetCerts: () => Array<ArrayBuffer>;
    readonly GetCertsHash: () => ArrayBuffer;
    readonly GetState: () => ArrayBuffer;
    readonly SetState: (s: ArrayBuffer) => void;
    readonly GetInputs: () => Array<ArrayBuffer>;
    readonly SetOutputs: (outputs: any[] | ArrayBuffer) => void;
    readonly GetRootKey: () => Bip32Key;
}

export declare const GetEGVMContext: () => EGVMContext;
