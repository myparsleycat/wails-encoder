interface VideoFile {
  id: string;
  name: string;
  size: string;
  duration: string;
  format: string;
  status: '추가됨' | '진행중' | '완료' | '실패';
  path: string;  // File 대신 파일 경로 저장
}