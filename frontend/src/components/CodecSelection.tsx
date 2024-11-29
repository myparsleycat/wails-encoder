// encoder/frontend/src/components/CodecSelection.tsx

import React, { useEffect, useState } from 'react';
import { Select, SelectItem } from "@nextui-org/react";
import { GetAvailableCodecs } from '../../wailsjs/go/main/App';
import { toast } from 'sonner';

interface CodecInfo {
  name: string;
  displayName: string;
  hardware: string;
  formats: string[];
}

interface CodecSelectionProps {
  selectedFormat: string;
  selectedCodec: string;  // 추가
  onCodecChange: (codec: string) => void;
}

const CodecSelection: React.FC<CodecSelectionProps> = ({
  selectedFormat,
  selectedCodec: initialSelectedCodec,  // 이름 변경
  onCodecChange
}) => {
  const [availableCodecs, setAvailableCodecs] = useState<CodecInfo[]>([]);
  const [selectedCodec, setSelectedCodec] = useState(initialSelectedCodec);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const loadCodecs = async () => {
      try {
        const codecs = await GetAvailableCodecs();
        setAvailableCodecs(codecs);

        // 이미 선택된 코덱이 없을 때만 첫 번째 코덱을 선택
        if (!initialSelectedCodec) {
          const formatCodecs = codecs.filter(codec =>
            codec.formats.includes(selectedFormat)
          );
          if (formatCodecs.length > 0) {
            setSelectedCodec(formatCodecs[0].name);
            onCodecChange(formatCodecs[0].name);
          }
        }
      } catch (err: any) {
        toast.error('코덱 정보 로딩 실패', {
          description: err.message
        });
      } finally {
        setIsLoading(false);
      }
    };

    loadCodecs();
  }, [selectedFormat, initialSelectedCodec]);

  const formatCodecs = availableCodecs.filter(codec =>
    codec.formats.includes(selectedFormat)
  );

  return (
    <Select
      label="비디오 코덱"
      selectedKeys={[selectedCodec]}
      onChange={(e) => {
        setSelectedCodec(e.target.value);
        onCodecChange(e.target.value);
      }}
      isLoading={isLoading}
    >
      {formatCodecs.map((codec) => (
        <SelectItem
          key={codec.name}
          value={codec.name}
          description={`${codec.hardware === 'cpu' ? 'CPU 인코딩' : '하드웨어 가속'}`}
        >
          {codec.displayName}
        </SelectItem>
      ))}
    </Select>
  );
};

export default CodecSelection;