import {
  memo,
  useCallback,
  useEffect,
  useState
} from "react";

import {
  OnFileDrop,
  OnFileDropOff,
  EventsOn,
  EventsOff
} from "../wailsjs/runtime/runtime";

import {
  ProcessVideoPaths,
  StartEncodingWithOptions,
  ShowNotification
} from '../wailsjs/go/main/App';

import { toast } from "sonner";
import {
  Button,
  Table, TableHeader, TableColumn, TableBody, TableRow, TableCell,
  getKeyValue,
  Select, SelectItem,
  Input,
  Switch,
  Progress,
  Modal, ModalContent, ModalHeader, ModalBody, ModalFooter,
  useDisclosure,
  Tabs,
  Card, CardBody,
  Tab
} from "@nextui-org/react";
import CodecSelection from "./components/CodecSelection";


// 지원되는 포맷과 코덱 정의
const supportedFormats = {
  mp4: ["h264", "h264_nvenc", "h264_qsv", "hevc", "hevc_nvenc", "hevc_qsv"],
  webm: ["vp8", "vp9"],
};

// 코덱별 기본 설정
const codecDefaults = {
  h264: { mode: "crf", defaultValue: 23, min: 0, max: 51 },
  hevc: { mode: "crf", defaultValue: 28, min: 0, max: 51 },
  vp9: { mode: "crf", defaultValue: 31, min: 0, max: 63 },
};

const qualityModes = {
  crf: {
    label: "CRF (품질 기반)",
    description: "파일 크기와 관계없이 일정한 품질 유지"
  },
  bitrate: {
    label: "비트레이트 (크기 기반)",
    description: "특정 파일 크기를 목표로 인코딩"
  }
};

const columns = [
  { key: 'name', label: '파일명' },
  { key: 'size', label: '크기' },
  { key: 'duration', label: '길이' },
  { key: 'format', label: '형식' },
  { key: 'status', label: '상태' },
];

interface VideoFile {
  id: string;
  name: string;
  size: string;
  duration: string;
  format: string;
  status: '대기중' | '진행중' | '완료' | '실패';
  path: string;
  progress?: number;
  encodingInfo?: {
    frame: number;
    fps: number;
    time: string;
    size: number;
    bitrate: number;
    speed: number;
  };
}

interface EncodingProgress {
  filename: string;
  frame: number;
  fps: number;
  time: string;
  size: number;
  bitrate: number;
  speed: number;
  progress: number;
  status: string;
}


const formatDuration = (seconds: number) => {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};

const formatSize = (size: number) => {
  return size < 1024 * 1024 * 1024
    ? `${(size / 1024 / 1024).toFixed(2)}MB`
    : `${(size / 1024 / 1024 / 1024).toFixed(2)}GB`;
};

const VideoRow = memo(({ video, columnKey }: { video: VideoFile, columnKey: string }) => {
  if (columnKey === "status") {
    return (
      <div className="flex flex-col gap-2 whitespace-nowrap">
        <span className="whitespace-nowrap">{video.status}</span>
        {video.status === '진행중' && video.progress !== undefined && (
          <Progress
            size="sm"
            value={video.progress}
            color="primary"
            className="max-w-md"
          />
        )}
      </div>
    );
  }
  return <>{getKeyValue(video, columnKey)}</>;
});

export default function App() {
  const [selectedKeys, setSelectedKeys] = useState<Set<VideoFile['id']>>(new Set([]));
  const [videos, setVideos] = useState<VideoFile[]>([]);
  const [isAnalysing, setIsAnalysing] = useState(false);
  const [encodeProsessing, setEncodeProsessing] = useState(false);

  //
  const { isOpen, onOpen, onOpenChange } = useDisclosure();

  // 인코딩 옵션 상태
  const [videoFormat, setVideoFormat] = useState("mp4");
  const [videoCodec, setVideoCodec] = useState("h264");
  const [qualityMode, setQualityMode] = useState<"crf" | "bitrate">("crf");
  const [qualityValue, setQualityValue] = useState(23); // CRF 기본값
  const [bitrateValue, setBitrateValue] = useState(5000); // 5Mbps 기본값
  const [use2Pass, setUse2Pass] = useState(false);
  const [prefix, setPrefix] = useState("encoded_");
  const [postfix, setPostfix] = useState("");

  //
  const [outputpath, setOutputpath] = useState("");

  const [isResize, setIsResize] = useState(false);
  const [width, setWidth] = useState(0);
  const [height, setHeight] = useState(0);

  const [overallProgress, setOverallProgress] = useState(0);

  // 코덱 변경시 기본값 설정
  useEffect(() => {
    const codecDefault = codecDefaults[videoCodec as keyof typeof codecDefaults];
    if (codecDefault) {
      setQualityMode(codecDefault.mode as 'crf' | 'bitrate');
      if (codecDefault.mode === "crf") {
        setQualityValue(codecDefault.defaultValue);
      } else {
        setBitrateValue(5000); // 기본 비트레이트
      }
    }
  }, [videoCodec]);

  useEffect(() => {
    // 중복 등록 방지를 위한 flag
    let isSubscribed = true;

    // 비디오 처리 이벤트 리스너
    const handleVideoProcessed = (metadata: any) => {
      if (!isSubscribed) return;

      const newVideo: VideoFile = {
        id: Math.random().toString(36).substring(7),
        name: metadata.name,
        size: formatSize(metadata.size),
        duration: formatDuration(metadata.duration),
        format: `${metadata.format} (${metadata.codec})`,
        status: '대기중',
        path: metadata.path
      };

      setVideos(prev => {
        // 중복 체크
        if (prev.some(v => v.path === metadata.path)) {
          return prev;
        }
        return [...prev, newVideo];
      });

      setSelectedKeys(prev => {
        const newSet = new Set(prev);
        newSet.add(newVideo.id);
        return newSet;
      });
    };

    // 에러 처리 이벤트 리스너
    const handleVideoError = (error: any) => {
      if (!isSubscribed) return;
      toast.error('파일 처리 오류', { description: error.error });
    };

    const handleFileDrop = async (x: number, y: number, paths: string[]) => {
      if (isAnalysing) {
        toast.error('이미 파일을 처리중입니다.');
        return;
      }

      try {
        setIsAnalysing(true);
        await ProcessVideoPaths(paths);
      } catch (err: any) {
        toast.error('오류 발생', { description: err.message });
      } finally {
        setIsAnalysing(false);
      }
    };

    // 이벤트 리스너 등록
    EventsOn("video_processed", handleVideoProcessed);
    EventsOn("video_error", handleVideoError);
    OnFileDrop(handleFileDrop, true);

    // Cleanup function
    return () => {
      isSubscribed = false;
      OnFileDropOff();
      EventsOff("video_processed");
      EventsOff("video_error");
    };
  }, []);

  // 프로그레스 이벤트 리스너 설정
  useEffect(() => {
    const handleProgress = (progress: EncodingProgress) => {
      setVideos(prev => prev.map(video => {
        if (video.name === progress.filename) {
          // 진행 상태 업데이트
          const newStatus = progress.status as VideoFile['status'];
          // 시간 문자열에서 초 단위로 변환
          const currentTime = progress.time.split(':').reduce((acc, time) => (60 * acc) + parseFloat(time), 0);
          // 비디오 길이 문자열에서 초 단위로 변환
          const totalDuration = video.duration.split(':').reduce((acc, time) => (60 * acc) + parseFloat(time), 0);
          // 진행률 계산 (0-100)
          const calculatedProgress = (currentTime / totalDuration) * 100;

          return {
            ...video,
            status: newStatus,
            progress: Math.min(Math.max(calculatedProgress, 0), 100), // 0-100 범위로 제한
            encodingInfo: {
              frame: progress.frame,
              fps: progress.fps,
              time: progress.time,
              size: progress.size,
              bitrate: progress.bitrate,
              speed: progress.speed,
            }
          };
        }
        return video;
      }));

      // 전체 진행률 계산 수정
      setVideos(prev => {
        const totalVideos = prev.length;
        if (totalVideos === 0) return prev;

        const completedVideos = prev.filter(v => v.status === '완료').length;
        const inProgressVideos = prev.filter(v => v.status === '진행중');
        const progressSum = inProgressVideos.reduce((acc, v) => acc + (v.progress || 0), 0);

        const totalProgress = ((completedVideos * 100) + progressSum) / totalVideos;
        setOverallProgress(Math.min(Math.max(totalProgress, 0), 100)); // 0-100 범위로 제한

        return prev;
      });
    };

    EventsOn("encoding_progress", handleProgress);
    return () => EventsOff("encoding_progress");
  }, []);

  const handleEncoding = async () => {
    try {
      setOverallProgress(0);
      // 모든 비디오의 상태를 초기화
      setVideos(prev => prev.map(video => ({
        ...video,
        status: selectedKeys.has(video.id) ? '대기중' : video.status,
        progress: selectedKeys.has(video.id) ? 0 : video.progress,
        error: undefined
      })));

      // 선택된 비디오만 필터링
      const selectedVideos = videos.filter(video => selectedKeys.has(video.id));

      // 선택된 비디오가 없는 경우 처리
      if (selectedVideos.length === 0) {
        toast.warning('인코딩할 비디오를 선택해주세요.');
        return;
      }

      const options = {
        videoformat: videoFormat,
        videocodec: videoCodec,
        qualitymode: qualityMode,
        qualityvalue: qualityMode === "crf" ? qualityValue : bitrateValue,
        use2pass: qualityMode === "bitrate" && use2Pass,
        isresize: isResize,
        width: width,
        height: height,
        outputpath,
        prefix,
        postfix,
        audiocodec: "",
        audiobitrate: 0,
        audiosamplerate: 0
      };

      console.log("Starting encoding with options:", options);
      setEncodeProsessing(true);

      // 선택된 비디오의 경로만 전달
      const result = await StartEncodingWithOptions(selectedVideos.map(v => v.path), options);
      console.log("Encoding result:", result);

      toast.success('인코딩이 완료되었습니다.');
    } catch (error: any) {
      setOverallProgress(0);
      console.error("Encoding error:", error);

      const errorMessage = error?.message || error?.toString() || "알 수 없는 오류가 발생했습니다.";
      const errorDetails = error?.details || error?.stack || "";

      console.error("Error details:", {
        message: errorMessage,
        details: errorDetails,
        originalError: error
      });

      toast.error('인코딩 실패', {
        description: errorMessage,
        duration: 5000,
      });

      // 선택된 실패한 비디오의 상태만 업데이트
      setVideos(prev => prev.map(video => ({
        ...video,
        status: selectedKeys.has(video.id) && video.status === '진행중' ? '실패' : video.status
      })));
    } finally {
      setEncodeProsessing(false);
    }
  };

  const handleStopEncoding = () => {

  }

  const resetAll = () => {
    setSelectedKeys(new Set([]));
    setVideos([]);
    setIsAnalysing(false);
    setOverallProgress(0);
  }

  return (
    <div className="h-screen flex flex-col p-6">
      <div className="flex-1 min-h-0 relative mb-4" style={{ "--wails-drop-target": "drop" } as React.CSSProperties}>
        <div className="absolute inset-0 flex flex-col">
          <Table
            isHeaderSticky
            aria-label="video table"
            selectionMode="multiple"
            selectedKeys={selectedKeys}
            onSelectionChange={setSelectedKeys as any}
            className="h-full"
            classNames={{
              wrapper: "h-full",
              table: "h-full min-h-[200px] h-full",
              thead: "sticky top-0 z-10",
              tbody: "overflow-y-auto",
            }}
          >
            <TableHeader
              columns={columns}
            >
              {(column) => <TableColumn key={column.key}>{column.label}</TableColumn>}
            </TableHeader>
            <TableBody
              items={videos}
              emptyContent={
                "여기로 영상을 드래그 앤 드랍해 추가하세요"
              }
            >
              {(video) => (
                <TableRow key={video.id}>
                  {(columnKey) => (
                    <TableCell>
                      <VideoRow video={video} columnKey={columnKey as string} />
                    </TableCell>
                  )}
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="space-y-4 shrink-0">
        <div className="justify-between flex">
          <div className="space-x-4">
            <Button
              color="default"
              variant="ghost"
              isDisabled={selectedKeys.size < 1 || encodeProsessing}
              onPress={() => {
                const selectedKeysArray = Array.from(selectedKeys);
                setOverallProgress(0);
                setVideos(prev => prev.filter(video => !selectedKeysArray.includes(video.id)));
                setSelectedKeys(new Set([]));
              }}
            >
              선택 제거
            </Button>
            <Button
              color="default"
              variant="ghost"
              isDisabled={videos.length < 1 || encodeProsessing}
              onClick={resetAll}
            >
              전체 제거
            </Button>
          </div>
          <div className="space-x-2">
            <Button
              color="default"
              variant="ghost"
              onPress={onOpen}
              isDisabled={encodeProsessing}
            >인코딩 설정
            </Button>

            <Button
              color="default"
              variant="ghost"
              onClick={handleStopEncoding}
              isDisabled={!encodeProsessing}
            >
              인코딩 취소
            </Button>

            <Button
              color="default"
              variant="ghost"
              isDisabled={videos.length < 1 || encodeProsessing}
              onClick={handleEncoding}
            >
              인코딩 시작
            </Button>
          </div>
        </div>

        <div className="w-full">
          <Progress
            aria-label="overall-progress-bar"
            value={overallProgress}
            className="max-w-full"
            color="primary"
            showValueLabel={true}
          />
          <div className="text-sm text-center mt-1">
            전체 진행률: {Math.round(overallProgress)}%
          </div>
        </div>
      </div>

      <Modal
        size="full"
        isOpen={isOpen}
        onOpenChange={onOpenChange}
        closeButton={<></>}
      >
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader className="flex flex-col gap-1"></ModalHeader>
              <ModalBody>
                <div className="flex flex-col">
                  <Tabs aria-label="Options" isVertical={true}>
                    <Tab key="encoding" title="인코딩" className="w-full">
                      <Card>
                        <CardBody className="gap-4">
                          <div className="grid grid-cols-2 gap-4">
                            <Select
                              label="출력 포맷"
                              selectedKeys={[videoFormat]}
                              onChange={(e) => setVideoFormat(e.target.value)}
                            >
                              {Object.keys(supportedFormats).map((format) => (
                                <SelectItem key={format} value={format}>
                                  {format.toUpperCase()}
                                </SelectItem>
                              ))}
                            </Select>

                            <CodecSelection
                              selectedFormat={videoFormat}
                              selectedCodec={videoCodec}
                              onCodecChange={(codec) => setVideoCodec(codec)}
                            />
                          </div>

                          <div className="grid grid-cols-2 gap-4">
                            <Select
                              label="인코딩 모드"
                              selectedKeys={[qualityMode]}
                              onChange={(e) => setQualityMode(e.target.value as "crf" | "bitrate")}
                              description={qualityModes[qualityMode].description}
                            >
                              {Object.entries(qualityModes).map(([key, value]) => (
                                <SelectItem key={key} value={key}>
                                  {value.label}
                                </SelectItem>
                              ))}
                            </Select>

                            {qualityMode === "crf" ? (
                              <Input
                                type="number"
                                label="CRF 값"
                                description={`품질 값 (${codecDefaults[videoCodec as keyof typeof codecDefaults]?.min || 0}-${codecDefaults[videoCodec as keyof typeof codecDefaults]?.max || 51})`}
                                value={qualityValue.toString()}
                                onChange={(e) => {
                                  const value = parseInt(e.target.value);
                                  const codec = codecDefaults[videoCodec as keyof typeof codecDefaults];
                                  // if (codec && value >= codec.min && value <= codec.max) {
                                  setQualityValue(value);
                                  // }
                                }}
                              />
                            ) : (
                              <div className="grid grid-cols-2 gap-4">
                                <Input
                                  type="number"
                                  label="비트레이트 (kbps)"
                                  description="목표 비트레이트 (예: 5000 = 5Mbps)"
                                  value={bitrateValue.toString()}
                                  onChange={(e) => setBitrateValue(parseInt(e.target.value))}
                                />

                                <div className="flex items-center space-x-2">
                                  <Switch
                                    checked={use2Pass}
                                    onChange={(e) => setUse2Pass(e.target.checked)}
                                  />
                                  <div>
                                    <p>2-Pass 인코딩</p>
                                    <p className="text-sm text-gray-500">더 정확한 비트레이트 제어 (시간 증가)</p>
                                  </div>
                                </div>
                              </div>
                            )}
                          </div>

                          <div className="flex flex-col gap-4 mt-4">
                            <div className="flex items-center space-x-2">
                              <Switch
                                checked={isResize}
                                onChange={(e) => setIsResize(e.target.checked)}
                              />
                              <span>크기 조정</span>
                            </div>

                            {isResize && (
                              <div className="grid grid-cols-2 gap-4">
                                <Input
                                  type="number"
                                  label="너비"
                                  value={width.toString()}
                                  onChange={(e) => setWidth(parseInt(e.target.value))}
                                />
                                <Input
                                  type="number"
                                  label="높이"
                                  value={height.toString()}
                                  onChange={(e) => setHeight(parseInt(e.target.value))}
                                />
                              </div>
                            )}
                          </div>
                        </CardBody>
                      </Card>
                    </Tab>
                    <Tab key="audio" title="오디오">
                      <Card>
                        <CardBody>

                        </CardBody>
                      </Card>
                    </Tab>
                    <Tab key="other" title="기타" className="w-full">
                      <Card>
                        <CardBody className="gap-4">
                          <div className="grid grid-cols-2 gap-4">
                            <Input
                              label="접두사"
                              description="encoded_파일이름"
                              value={prefix}
                              onChange={(e) => setPrefix(e.target.value)}
                            />

                            <Input
                              label="접미사"
                              description="파일이름_encoded"
                              value={postfix}
                              onChange={(e) => setPostfix(e.target.value)}
                            />
                          </div>
                        </CardBody>
                      </Card>
                    </Tab>
                  </Tabs>
                </div>
              </ModalBody>
              <ModalFooter>
                <Button color="primary" onPress={onClose}>
                  확인
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}