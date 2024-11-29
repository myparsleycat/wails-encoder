// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {main} from '../models';

export function EmitProgress(arg1:main.EncodingProgress):Promise<void>;

export function FindVideoFiles(arg1:string):Promise<Array<string>>;

export function GetAvailableCodecs():Promise<Array<main.CodecInfo>>;

export function ProcessVideo(arg1:string):Promise<main.VideoMetadata>;

export function ProcessVideoPaths(arg1:Array<string>):Promise<void>;

export function ShowNotification(arg1:string,arg2:string):Promise<void>;

export function StartEncodingWithOptions(arg1:Array<string>,arg2:main.EncodingOptions):Promise<void>;
