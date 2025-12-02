import { Transition, Variants } from 'framer-motion';

type Easing = [number, number, number, number];

export const motionConfig = {
  duration: 0.3,
  durationFast: 0.18,
  ease: [0.22, 1, 0.36, 1] as Easing,
  stagger: 0.05,
};

export const layoutTransition: Transition = {
  type: 'spring',
  stiffness: 420,
  damping: 38,
};

export const fadeIn: Variants = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: { duration: motionConfig.durationFast, ease: motionConfig.ease },
  },
  exit: {
    opacity: 0,
    transition: { duration: motionConfig.durationFast, ease: motionConfig.ease },
  },
};

export const slideUp: Variants = {
  hidden: { opacity: 0, y: 18 },
  show: {
    opacity: 1,
    y: 0,
    transition: { duration: motionConfig.duration, ease: motionConfig.ease },
  },
  exit: {
    opacity: 0,
    y: 12,
    transition: { duration: motionConfig.duration, ease: motionConfig.ease },
  },
};

export const staggerContainer: Variants = {
  hidden: {},
  show: {
    transition: {
      staggerChildren: motionConfig.stagger,
      delayChildren: 0.06,
    },
  },
};
