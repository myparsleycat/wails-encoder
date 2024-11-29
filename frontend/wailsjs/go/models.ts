export namespace main {
	
	export class CodecInfo {
	    name: string;
	    displayName: string;
	    hardware: string;
	    formats: string[];
	
	    static createFrom(source: any = {}) {
	        return new CodecInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.hardware = source["hardware"];
	        this.formats = source["formats"];
	    }
	}
	export class EncodingOptions {
	    videoformat: string;
	    videocodec: string;
	    qualitymode: string;
	    qualityvalue: number;
	    use2pass: boolean;
	    isresize: boolean;
	    width: number;
	    height: number;
	    outputpath: string;
	    prefix: string;
	    postfix: string;
	    audiocodec: string;
	    audiobitrate: number;
	    audiosamplerate: number;
	
	    static createFrom(source: any = {}) {
	        return new EncodingOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.videoformat = source["videoformat"];
	        this.videocodec = source["videocodec"];
	        this.qualitymode = source["qualitymode"];
	        this.qualityvalue = source["qualityvalue"];
	        this.use2pass = source["use2pass"];
	        this.isresize = source["isresize"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.outputpath = source["outputpath"];
	        this.prefix = source["prefix"];
	        this.postfix = source["postfix"];
	        this.audiocodec = source["audiocodec"];
	        this.audiobitrate = source["audiobitrate"];
	        this.audiosamplerate = source["audiosamplerate"];
	    }
	}
	export class EncodingProgress {
	    filename: string;
	    frame: number;
	    fps: number;
	    time: string;
	    size: number;
	    bitrate: number;
	    speed: number;
	    progress: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new EncodingProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.frame = source["frame"];
	        this.fps = source["fps"];
	        this.time = source["time"];
	        this.size = source["size"];
	        this.bitrate = source["bitrate"];
	        this.speed = source["speed"];
	        this.progress = source["progress"];
	        this.status = source["status"];
	    }
	}
	export class VideoMetadata {
	    name: string;
	    size: number;
	    duration: number;
	    format: string;
	    codec: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new VideoMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.duration = source["duration"];
	        this.format = source["format"];
	        this.codec = source["codec"];
	        this.path = source["path"];
	    }
	}

}

