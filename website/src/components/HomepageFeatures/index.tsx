import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Single dependency-free script',
    description: (
      <>
        No PHP, Composer, or runtime to install — <code>lpod</code> is one
        bash script you can curl down or clone and run directly.
      </>
    ),
  },
  {
    title: 'Wraps Quadlet, systemctl & podman exec',
    description: (
      <>
        Start, stop, and inspect Quadlet-managed services, with unrecognized
        commands passing straight through to <code>podman</code> itself.
      </>
    ),
  },
  {
    title: 'Works standalone',
    description: (
      <>
        Pairs naturally with <code>foxws/laravel-podman</code>, but{' '}
        <code>lpod</code> doesn&apos;t require it — point it at any
        Quadlet-managed service.
      </>
    ),
  },
];

function Feature({title, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
