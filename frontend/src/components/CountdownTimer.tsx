import React, { useState, useEffect } from 'react';
import { CountdownTime } from '../types';

interface CountdownTimerProps {
  targetTime: string;
  onComplete?: () => void;
  prefix?: string;
  className?: string;
}

const CountdownTimer: React.FC<CountdownTimerProps> = ({ 
  targetTime, 
  onComplete, 
  prefix = "距离秒杀开始还有：",
  className = ""
}) => {
  const [timeLeft, setTimeLeft] = useState<CountdownTime>({
    days: 0,
    hours: 0,
    minutes: 0,
    seconds: 0,
  });
  const [isCompleted, setIsCompleted] = useState(false);

  useEffect(() => {
    const calculateTimeLeft = () => {
      const now = new Date().getTime();
      const target = new Date(targetTime).getTime();
      const difference = target - now;

      if (difference <= 0) {
        setTimeLeft({ days: 0, hours: 0, minutes: 0, seconds: 0 });
        if (!isCompleted) {
          setIsCompleted(true);
          onComplete?.();
        }
        return;
      }

      const days = Math.floor(difference / (1000 * 60 * 60 * 24));
      const hours = Math.floor((difference % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
      const minutes = Math.floor((difference % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((difference % (1000 * 60)) / 1000);

      setTimeLeft({ days, hours, minutes, seconds });
    };

    calculateTimeLeft();
    const timer = setInterval(calculateTimeLeft, 1000);

    return () => clearInterval(timer);
  }, [targetTime, onComplete, isCompleted]);

  const formatNumber = (num: number): string => {
    return num.toString().padStart(2, '0');
  };

  if (isCompleted) {
    return (
      <div className={`text-center ${className}`}>
        <div className="text-green-600 font-bold text-lg animate-pulse">
          🎉 秒杀已开始！
        </div>
      </div>
    );
  }

  return (
    <div className={`text-center ${className}`}>
      <div className="text-gray-700 text-sm mb-2 font-medium">
        {prefix}
      </div>
      <div className="flex justify-center items-center space-x-2">
        {timeLeft.days > 0 && (
          <>
            <div className="countdown-digit">
              {formatNumber(timeLeft.days)}
            </div>
            <span className="text-gray-500 font-medium">天</span>
          </>
        )}
        
        <div className="countdown-digit">
          {formatNumber(timeLeft.hours)}
        </div>
        <span className="text-gray-500 font-medium">:</span>
        
        <div className="countdown-digit">
          {formatNumber(timeLeft.minutes)}
        </div>
        <span className="text-gray-500 font-medium">:</span>
        
        <div className="countdown-digit animate-pulse">
          {formatNumber(timeLeft.seconds)}
        </div>
      </div>
      
      <div className="flex justify-center items-center space-x-4 mt-2 text-xs text-gray-500">
        <span>时</span>
        <span>分</span>
        <span>秒</span>
      </div>
    </div>
  );
};

export default CountdownTimer; 